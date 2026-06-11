import client from "./client";

/* GET /api/vin/:vin */
export const getVehicle = (vin) => client.get(`/vin/${vin}`);

/* GET /api/id/:id */
export const getVehicleById = (id) => client.get(`/id/${id}`);

/* PATCH /api/update/:vin */
export const updateVehicle = (vin, fields) =>
  client.patch(`/update/${vin}`, fields);

/* GET /api/gm/decode/:vin
   Live RPO / build-option lookup from GM Parts Giant for a specific full VIN.
   Data is NOT persisted — call this on-demand from the Vehicle page. */
export const fetchGMLive = (vin) => client.get(`/gm/decode/${vin}`);
