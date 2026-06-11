import { Routes, Route, Navigate } from "react-router-dom";
import { useAuth } from "./contexts/AuthContext";
import AuthPage from "./pages/AuthPage";
import SearchPage from "./pages/SearchPage";
import VehiclePage from "./pages/VehiclePage";
import ListingsPage from "./pages/ListingsPage";
import AdminPage from "./pages/AdminPage";
import HistoryPage from "./pages/HistoryPage";
import ListingErrorsPage from "./pages/ListingErrorsPage";
import DNRPage from "./pages/DNRPage";
import PartsPage from "./pages/PartsPage";
import ImportPage from "./pages/ImportPage";

function ProtectedRoute({ children }) {
  const { token } = useAuth();
  return token ? children : <Navigate to="/auth" replace />;
}

function AdminRoute({ children }) {
  const { token, user } = useAuth();
  if (!token) return <Navigate to="/auth" replace />;
  if (!user?.isAdmin) return <Navigate to="/" replace />;
  return children;
}

/* Admin OR listing role — for pages the listing team also needs */
function StaffRoute({ children }) {
  const { token, user } = useAuth();
  if (!token) return <Navigate to="/auth" replace />;
  if (!user?.isAdmin && !user?.isListing) return <Navigate to="/" replace />;
  return children;
}

/* Admin OR dnr role */
function DNRRoute({ children }) {
  const { token, user } = useAuth();
  if (!token) return <Navigate to="/auth" replace />;
  if (!user?.isAdmin && !user?.isDNR) return <Navigate to="/" replace />;
  return children;
}

export default function App() {
  return (
    <Routes>
      <Route path="/auth" element={<AuthPage />} />

      <Route
        path="/"
        element={
          <ProtectedRoute>
            <SearchPage />
          </ProtectedRoute>
        }
      />

      <Route
        path="/v/:vin"
        element={
          <ProtectedRoute>
            <VehiclePage />
          </ProtectedRoute>
        }
      />

      <Route
        path="/listings"
        element={
          <ProtectedRoute>
            <ListingsPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/history"
        element={
          <StaffRoute>
            <HistoryPage />
          </StaffRoute>
        }
      />
      <Route
        path="/listing-error"
        element={
          <StaffRoute>
            <ListingErrorsPage />
          </StaffRoute>
        }
      />

      <Route
        path="/admin"
        element={
          <AdminRoute>
            <AdminPage />
          </AdminRoute>
        }
      />

      <Route
        path="/dnr"
        element={
          <DNRRoute>
            <DNRPage />
          </DNRRoute>
        }
      />

      {/* <Route
        path="/parts"
        element={
          <ProtectedRoute>
            <PartsPage />
          </ProtectedRoute>
        }
      />*/}

      <Route
        path="/import"
        element={
          <DNRRoute>
            <ImportPage />
          </DNRRoute>
        }
      />

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
