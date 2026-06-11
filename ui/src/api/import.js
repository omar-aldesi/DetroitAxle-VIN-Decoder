import client from "./client";

export const startImport = (payload) => client.post("/import/", payload);
export const getImportJob = (jobId) => client.get(`/import/${jobId}`).then((r) => r.data);
export const cancelImport = (jobId) => client.delete(`/import/${jobId}`);
