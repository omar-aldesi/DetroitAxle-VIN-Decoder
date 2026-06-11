package handlers

import (
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	sqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"main/auth"
	"main/models"
	"main/services"
)

// setupForkFlowDB builds a fresh DB matching the current Vehicle model (no legacy
// build-number columns) — the state every real DB is in after the fork-field columns
// were removed from models.Vehicle.
func setupForkFlowDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbFile := filepath.Join(t.TempDir(), "fork_flow.db")
	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Close the connection so Windows can remove the temp file during cleanup.
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { sqlDB.Close() })
	}
	if err := db.AutoMigrate(
		&models.Vehicle{}, &models.User{}, &models.AgentNote{}, &models.PartCategory{},
		&models.FieldPermission{}, &models.VehicleFieldHistory{},
		&models.ForkField{}, &models.FieldRange{}, &models.FieldPoint{}, &models.KnownVIN{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	if err := services.SeedDefaultForkFields(db); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return db
}

func ctxWith(method, body string, params gin.Params, user *models.User) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = params
	c.Set(auth.CurrentUserKey, user)
	return c, w
}

// TestUpdateVehicle_AgentEditVerifyFlow_CreatesForkRange exercises the path the user
// drives from the Vehicle page: an agent edits a build-number-tier (fork) field on two
// VINs that share a build key, each edit is verified, and a third VIN whose serial
// falls between the two should resolve the field with no pending points left over.
func TestUpdateVehicle_AgentEditVerifyFlow_CreatesForkRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupForkFlowDB(t)

	veh := models.Vehicle{
		BuildKey: "KMHE24L1HA",
		Year:     2017, Make: "Hyundai", Model: "Elantra GT",
	}
	if err := db.Create(&veh).Error; err != nil {
		t.Fatalf("create vehicle: %v", err)
	}

	agent := &models.User{Email: "agent@x.com", Username: "agent", Role: "agent", IsActive: true}
	admin := &models.User{Email: "admin@x.com", Username: "admin", Role: "admin", IsActive: true}
	if err := db.Create(agent).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}

	vh := &VehicleHandler{DB: db}
	hh := &HistoryHandler{DB: db}
	fh := &ForkHandler{DB: db}

	editAndVerify := func(vin string) {
		c, w := ctxWith("PATCH", `{"brake_code":"JP9"}`, gin.Params{{Key: "vin", Value: vin}}, agent)
		vh.UpdateVehicle(c)
		if w.Code != 200 {
			t.Fatalf("update %s -> %d %s", vin, w.Code, w.Body.String())
		}

		var h models.VehicleFieldHistory
		if err := db.Where("field_name = ?", "brake_code").Order("id DESC").First(&h).Error; err != nil {
			t.Fatalf("find history for %s: %v", vin, err)
		}
		if h.Tier != "build_number" {
			t.Fatalf("history entry for %s: tier = %q, want build_number", vin, h.Tier)
		}

		body := fmt.Sprintf(`{"verifier_id":%d}`, admin.ID)
		vc, vw := ctxWith("PATCH", body, gin.Params{{Key: "id", Value: fmt.Sprint(h.ID)}}, admin)
		hh.VerifyEntry(vc)
		if vw.Code != 200 {
			t.Fatalf("verify %d -> %d %s", h.ID, vw.Code, vw.Body.String())
		}
	}

	// VIN1 (serial 1) and VIN2 (serial 100) both get brake_code=JP9, agreeing on the value.
	editAndVerify("KMHE24L14HA000001")
	editAndVerify("KMHE24L16HA000100")

	var ranges []models.FieldRange
	db.Find(&ranges)
	if len(ranges) != 1 {
		t.Fatalf("expected 1 fork range after two agreeing verified edits, got %d: %+v", len(ranges), ranges)
	}
	if ranges[0].Value != "JP9" || ranges[0].SerialStart != 1 || ranges[0].SerialEnd == nil || *ranges[0].SerialEnd != 100 {
		t.Fatalf("unexpected range: %+v", ranges[0])
	}

	// VIN3 (serial 99) sits inside [1,100] — it should resolve to JP9 with nothing pending.
	c, w := ctxWith("GET", "", gin.Params{{Key: "vin", Value: "KMHE24L13HA000099"}}, agent)
	fh.GetForkData(c)
	if w.Code != 200 {
		t.Fatalf("fork data for VIN3 -> %d %s", w.Code, w.Body.String())
	}

	pending, err := services.PendingPointsFor(db, "KMHE24L1HA")
	if err != nil {
		t.Fatalf("PendingPointsFor: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected no pending points after range creation, got %v", pending)
	}
}

// TestVerifyEntry_CorrectedValueOnForkField_DoesNotTouchVehiclesTable exercises the
// "Correct value" path in the History verify dialog for a build-number-tier (fork)
// field. brake_code etc. are no longer columns on `vehicles`, so the corrected value
// must flow only through the fork engine, not a raw `UPDATE vehicles SET brake_code=...`.
func TestVerifyEntry_CorrectedValueOnForkField_DoesNotTouchVehiclesTable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupForkFlowDB(t)

	veh := models.Vehicle{
		BuildKey: "KMHE24L1HA",
		Year:     2017, Make: "Hyundai", Model: "Elantra GT",
	}
	if err := db.Create(&veh).Error; err != nil {
		t.Fatalf("create vehicle: %v", err)
	}

	agent := &models.User{Email: "agent@x.com", Username: "agent", Role: "agent", IsActive: true}
	admin := &models.User{Email: "admin@x.com", Username: "admin", Role: "admin", IsActive: true}
	if err := db.Create(agent).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}

	vh := &VehicleHandler{DB: db}
	hh := &HistoryHandler{DB: db}

	c, w := ctxWith("PATCH", `{"brake_code":"JP9"}`, gin.Params{{Key: "vin", Value: "KMHE24L14HA000001"}}, agent)
	vh.UpdateVehicle(c)
	if w.Code != 200 {
		t.Fatalf("update -> %d %s", w.Code, w.Body.String())
	}

	var h models.VehicleFieldHistory
	if err := db.Where("field_name = ?", "brake_code").Order("id DESC").First(&h).Error; err != nil {
		t.Fatalf("find history: %v", err)
	}

	body := fmt.Sprintf(`{"verifier_id":%d,"corrected_value":"JP6"}`, admin.ID)
	vc, vw := ctxWith("PATCH", body, gin.Params{{Key: "id", Value: fmt.Sprint(h.ID)}}, admin)
	hh.VerifyEntry(vc)
	if vw.Code != 200 {
		t.Fatalf("verify with corrected_value -> %d %s", vw.Code, vw.Body.String())
	}

	var pts []models.FieldPoint
	db.Find(&pts)
	if len(pts) != 1 || pts[0].Value != "JP6" {
		t.Fatalf("expected one pending point with corrected value JP6, got %+v", pts)
	}
}

// TestUpdateVehicle_AdminForkEdit_DoesNotForkBeforeVerification guards the rule that
// vehicle-page fork edits never reach the range engine until verified — even for an
// admin. Previously admin/DNR edits fed the engine immediately, so a second VIN formed a
// range before its own edit was verified. Now an unverified edit must leave the engine
// untouched (only a pending history entry), and the range must form only on verification.
func TestUpdateVehicle_AdminForkEdit_DoesNotForkBeforeVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupForkFlowDB(t)

	veh := models.Vehicle{
		BuildKey: "KMHE24L1HA",
		Year:     2017, Make: "Hyundai", Model: "Elantra GT",
	}
	if err := db.Create(&veh).Error; err != nil {
		t.Fatalf("create vehicle: %v", err)
	}

	admin := &models.User{Email: "admin@x.com", Username: "admin", Role: "admin", IsActive: true}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}

	vh := &VehicleHandler{DB: db}
	hh := &HistoryHandler{DB: db}

	edit := func(vin string) {
		c, w := ctxWith("PATCH", `{"brake_code":"JP9"}`, gin.Params{{Key: "vin", Value: vin}}, admin)
		vh.UpdateVehicle(c)
		if w.Code != 200 {
			t.Fatalf("admin update %s -> %d %s", vin, w.Code, w.Body.String())
		}
	}

	// Admin edits both VINs on the vehicle page. Neither is verified yet.
	edit("KMHE24L14HA000001") // serial 1
	edit("KMHE24L16HA000100") // serial 100

	// The engine must be completely untouched: no points, no ranges.
	var pts []models.FieldPoint
	db.Find(&pts)
	if len(pts) != 0 {
		t.Fatalf("admin edits must not write fork points before verification, got %+v", pts)
	}
	var ranges []models.FieldRange
	db.Find(&ranges)
	if len(ranges) != 0 {
		t.Fatalf("a range must not form from unverified admin edits, got %+v", ranges)
	}

	// Verifying the first VIN records a single pending sighting — still no range.
	verify := func(serial int64) {
		var h models.VehicleFieldHistory
		if err := db.Where("field_name = ? AND origin_serial = ?", "brake_code", serial).
			Order("id DESC").First(&h).Error; err != nil {
			t.Fatalf("find history for serial %d: %v", serial, err)
		}
		body := fmt.Sprintf(`{"verifier_id":%d}`, admin.ID)
		vc, vw := ctxWith("PATCH", body, gin.Params{{Key: "id", Value: fmt.Sprint(h.ID)}}, admin)
		hh.VerifyEntry(vc)
		if vw.Code != 200 {
			t.Fatalf("verify serial %d -> %d %s", serial, vw.Code, vw.Body.String())
		}
	}

	verify(1)
	db.Find(&ranges)
	if len(ranges) != 0 {
		t.Fatalf("one verified VIN must stay a pending sighting, not a range, got %+v", ranges)
	}
	pending, err := services.PendingPointsFor(db, "KMHE24L1HA")
	if err != nil {
		t.Fatalf("PendingPointsFor: %v", err)
	}
	if got := pending["brake_code"]; len(got) != 1 || got[0].Serial != 1 || got[0].Value != "JP9" {
		t.Fatalf("expected one pending sighting #1=JP9, got %+v", got)
	}

	// Verifying the second agreeing VIN now confirms the range [1,100].
	verify(100)
	db.Find(&ranges)
	if len(ranges) != 1 {
		t.Fatalf("expected the range to form only after the second verification, got %+v", ranges)
	}
	if ranges[0].Value != "JP9" || ranges[0].SerialStart != 1 ||
		ranges[0].SerialEnd == nil || *ranges[0].SerialEnd != 100 {
		t.Fatalf("unexpected range: %+v", ranges[0])
	}
}
