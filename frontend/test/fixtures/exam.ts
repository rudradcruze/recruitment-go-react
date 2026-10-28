import type { components } from "../../src/api/schema";

export type Exam = components["schemas"]["Exam"];
export type ExamRequest = components["schemas"]["ExamRequest"];
export type ExamSummary = components["schemas"]["ExamSummary"];
export type AttemptResult = components["schemas"]["AttemptResult"];

let counter = 0;

function uuid(n: number): string {
  return `0198f2b0-0000-7000-8000-${String(n).padStart(12, "0")}`;
}

// A builder gives a valid object with default values, and each call gives unique values for the unique fields.
export function exam(overrides: Partial<Exam> = {}): Exam {
  counter += 1;
  return {
    id: uuid(counter),
    title: `Exam ${counter}`,
    questionIds: [],
    opensAt: "2026-11-01T03:00:00Z",
    closesAt: "2026-11-01T11:00:00Z",
    timeLimitMinutes: 60,
    linkToken: `token${String(counter).padStart(17, "0")}`,
    createdAt: "2026-10-27T10:00:00Z",
    ...overrides,
  };
}

export function examSummary(overrides: Partial<ExamSummary> = {}): ExamSummary {
  counter += 1;
  return {
    id: uuid(counter),
    title: `Exam ${counter}`,
    opensAt: "2026-11-01T03:00:00Z",
    closesAt: "2026-11-01T11:00:00Z",
    timeLimitMinutes: 60,
    questionCount: 2,
    createdAt: "2026-10-27T10:00:00Z",
    ...overrides,
  };
}

export function attemptResult(overrides: Partial<AttemptResult> = {}): AttemptResult {
  counter += 1;
  return {
    name: `Candidate ${counter}`,
    emailAddress: `candidate${counter}@example.com`,
    status: "submitted",
    score: { correct: 1, total: 2 },
    startedAt: "2026-11-01T03:10:00Z",
    finishedAt: "2026-11-01T03:40:00Z",
    ...overrides,
  };
}
