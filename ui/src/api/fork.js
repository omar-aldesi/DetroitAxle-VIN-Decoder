import client from "./client";

/* Build-number fork/range system.
   Build-key-tier facts are shared; these fields vary by individual VIN (serial). */

/* GET /api/fork/:vin  (17-char VIN or 10-char build key)
   → { build_key, fork_fields, fields:{field:[ranges]}, pending:{field:[serials]},
       serial?, resolved?:{field:{value, confidence, observations, ...}} } */
export const getForkData = (vinOrKey) => client.get(`/fork/${vinOrKey}`);

/* POST /api/fork/:vin/point  { field_key, value }   — DNR/admin only; writes directly
   to the fork engine (full 17-char VIN required; two-point rule applies) */
export const recordForkPoint = (vin, body) =>
  client.post(`/fork/${vin}/point`, body);

/* POST /api/fork/:vin/range  { field_key, value, serial_start, serial_end|null }
   — DNR/admin only; authoritative manual span (no verification, no two-point rule) */
export const recordForkRange = (vinOrKey, body) =>
  client.post(`/fork/${vinOrKey}/range`, body);
