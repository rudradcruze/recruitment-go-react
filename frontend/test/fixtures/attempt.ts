import type { components } from "../../src/api/schema";

export type ExamLink = components["schemas"]["ExamLink"];
export type Attempt = components["schemas"]["Attempt"];
export type CandidateQuestion = components["schemas"]["CandidateQuestion"];
export type StartAttemptRequest = components["schemas"]["StartAttemptRequest"];
export type SaveAnswerRequest = components["schemas"]["SaveAnswerRequest"];
export type Score = components["schemas"]["Score"];

let counter = 0;

function uuid(n: number): string {
  return `0198f2c0-0000-4000-8000-${String(n).padStart(12, "0")}`;
}

export function examLink(overrides: Partial<ExamLink> = {}): ExamLink {
  counter += 1;
  return {
    title: `Exam ${counter}`,
    opensAt: "2026-11-01T03:00:00Z",
    closesAt: "2026-11-01T11:00:00Z",
    timeLimitMinutes: 30,
    questionCount: 2,
    ...overrides,
  };
}

// candidateQuestion gives a Question with 3 Options, as a Candidate sees it: without the correct mark.
export function candidateQuestion(overrides: Partial<CandidateQuestion> = {}): CandidateQuestion {
  counter += 1;
  const n = counter * 10;
  return {
    id: uuid(n),
    text: `Question ${counter}?`,
    options: [
      { id: uuid(n + 1), text: `Option A of ${counter}` },
      { id: uuid(n + 2), text: `Option B of ${counter}` },
      { id: uuid(n + 3), text: `Option C of ${counter}` },
    ],
    ...overrides,
  };
}

// attempt gives an open Attempt that expires in 30 minutes, with 2 Questions and no Answer.
export function attempt(overrides: Partial<Attempt> = {}): Attempt {
  counter += 1;
  return {
    id: uuid(counter),
    status: "open",
    examTitle: `Exam ${counter}`,
    expiresAt: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
    questions: [candidateQuestion(), candidateQuestion()],
    answers: [],
    ...overrides,
  };
}
