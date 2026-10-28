import type { components } from "../../src/api/schema";

export type Admin = components["schemas"]["Admin"];
export type LoginRequest = components["schemas"]["LoginRequest"];

let counter = 0;

// A builder gives a valid object with default values, and each call gives unique values for the unique fields.
export function admin(overrides: Partial<Admin> = {}): Admin {
  counter += 1;
  return {
    id: `0198f2a0-0000-7000-8000-${String(counter).padStart(12, "0")}`,
    emailAddress: `admin${counter}@example.com`,
    ...overrides,
  };
}

export function loginRequest(overrides: Partial<LoginRequest> = {}): LoginRequest {
  counter += 1;
  return {
    emailAddress: `admin${counter}@example.com`,
    password: `password-of-admin-${counter}`,
    ...overrides,
  };
}
