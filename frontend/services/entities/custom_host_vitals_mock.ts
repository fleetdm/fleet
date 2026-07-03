// TODO(#48559): replace mock with live API. This module exists only so the
// Custom host vitals tab is demoable before the backend CRUD endpoints exist.
// It mirrors the shape of `services/entities/custom_host_vitals` (real service)
// so consumers can swap the import once the API lands.
import {
  ICustomHostVital,
  ICustomHostVitalFormData,
} from "interfaces/custom_host_vitals";
import { getTokenFromVitalId } from "pages/ManageControlsPage/Variables/cards/CustomHostVitalsTab/CustomHostVitalsTableConfig";

import {
  IListCustomHostVitalsApiParams,
  IListCustomHostVitalsResponse,
} from "./custom_host_vitals";

const nowISO = () => new Date().toISOString();

// TODO(#48559): replace mock with live API. In-memory store seeded with sample
// definitions so list/add/edit/delete behave against real React Query hooks.
let _mockCustomHostVitals: ICustomHostVital[] = [
  {
    id: 1,
    name: "Asset tag",
    created_at: nowISO(),
    updated_at: nowISO(),
  },
  {
    id: 2,
    name: "Department",
    created_at: nowISO(),
    updated_at: nowISO(),
  },
  {
    id: 3,
    name: "Purchase date",
    created_at: nowISO(),
    updated_at: nowISO(),
  },
];

let _nextId = 4;

const MOCK_LATENCY_MS = 300;

const delay = <T>(value: T): Promise<T> =>
  new Promise((resolve) => setTimeout(() => resolve(value), MOCK_LATENCY_MS));

export default {
  getCustomHostVitals(
    params: IListCustomHostVitalsApiParams
  ): Promise<IListCustomHostVitalsResponse> {
    const query = (params.query ?? "").trim().toLowerCase();
    // Mirror the intended backend behavior: `query` matches the vital name OR
    // its variable token (`$FLEET_HOST_VITAL_<id>`), case-insensitive substring.
    const filtered = query
      ? _mockCustomHostVitals.filter(
          (vital) =>
            vital.name.toLowerCase().includes(query) ||
            getTokenFromVitalId(vital.id).toLowerCase().includes(query)
        )
      : _mockCustomHostVitals;

    return delay({
      custom_host_vitals: filtered,
      count: filtered.length,
      meta: {
        has_next_results: false,
        has_previous_results: false,
      },
    });
  },

  addCustomHostVital(vital: ICustomHostVitalFormData) {
    const trimmed = vital.name.trim();
    const isDuplicate = _mockCustomHostVitals.some(
      (existing) => existing.name.toLowerCase() === trimmed.toLowerCase()
    );
    if (isDuplicate) {
      // Mimic the backend 409 the real add modal handles.
      return Promise.reject({ status: 409 });
    }
    const created: ICustomHostVital = {
      id: _nextId,
      name: trimmed,
      created_at: nowISO(),
      updated_at: nowISO(),
    };
    _nextId += 1;
    _mockCustomHostVitals = [..._mockCustomHostVitals, created];
    return delay(created);
  },

  updateCustomHostVital(id: number, vital: ICustomHostVitalFormData) {
    const trimmed = vital.name.trim();
    const isDuplicate = _mockCustomHostVitals.some(
      (existing) =>
        existing.id !== id &&
        existing.name.toLowerCase() === trimmed.toLowerCase()
    );
    if (isDuplicate) {
      return Promise.reject({ status: 409 });
    }
    _mockCustomHostVitals = _mockCustomHostVitals.map((existing) =>
      existing.id === id
        ? { ...existing, name: trimmed, updated_at: nowISO() }
        : existing
    );
    return delay({ success: true });
  },

  deleteCustomHostVital(id: number) {
    _mockCustomHostVitals = _mockCustomHostVitals.filter(
      (existing) => existing.id !== id
    );
    return delay({ success: true });
  },
};
