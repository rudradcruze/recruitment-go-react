import type { components } from "../../src/api/schema";

export type Question = components["schemas"]["Question"];
export type QuestionRequest = components["schemas"]["QuestionRequest"];

let counter = 0;

function uuid(n: number): string {
  return `0198f2a0-0000-7000-8000-${String(n).padStart(12, "0")}`;
}

// A builder gives a valid object with default values, and each call gives unique values for the unique fields.
// The Question has 4 Options, and the first one is correct.
export function question(overrides: Partial<Question> = {}): Question {
  counter += 1;
  const n = counter * 10;
  return {
    id: uuid(n),
    text: `Question ${counter}?`,
    options: [
      { id: uuid(n + 1), text: `Right answer ${counter}`, correct: true },
      { id: uuid(n + 2), text: "Wrong answer 1", correct: false },
      { id: uuid(n + 3), text: "Wrong answer 2", correct: false },
      { id: uuid(n + 4), text: "Wrong answer 3", correct: false },
    ],
    createdAt: "2026-10-27T10:00:00Z",
    ...overrides,
  };
}

export function questionRequest(overrides: Partial<QuestionRequest> = {}): QuestionRequest {
  counter += 1;
  return {
    text: `Question ${counter}?`,
    options: [
      { text: "Right", correct: true },
      { text: "Wrong", correct: false },
    ],
    ...overrides,
  };
}
