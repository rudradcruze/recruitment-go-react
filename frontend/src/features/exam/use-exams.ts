import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "../../api/client";
import { unwrap, unwrapEmpty } from "../../api/errors";
import type { components } from "../../api/schema";

export type Exam = components["schemas"]["Exam"];
export type ExamRequest = components["schemas"]["ExamRequest"];
export type AttemptResult = components["schemas"]["AttemptResult"];

// The keys of the queries start with the name of the context.
export const examsQueryKey = ["exam", "exams"] as const;
export const examQueryKey = (id: string) => ["exam", "exam", id] as const;
export const examAttemptsQueryKey = (id: string) => ["exam", "exam", id, "attempts"] as const;

export function useExams() {
  return useQuery({
    queryKey: examsQueryKey,
    queryFn: async () => unwrap(await api.GET("/admin/exams")),
  });
}

export function useExam(id: string | undefined) {
  return useQuery({
    queryKey: examQueryKey(id ?? ""),
    queryFn: async () =>
      unwrap(await api.GET("/admin/exams/{id}", { params: { path: { id: id ?? "" } } })),
    enabled: id !== undefined,
  });
}

export function useExamAttempts(id: string) {
  return useQuery({
    queryKey: examAttemptsQueryKey(id),
    queryFn: async () =>
      unwrap(await api.GET("/admin/exams/{id}/attempts", { params: { path: { id } } })),
  });
}

// useSaveExam creates an Exam, or changes the Exam with this ID.
export function useSaveExam(id: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: ExamRequest) =>
      id === undefined
        ? unwrap(await api.POST("/admin/exams", { body }))
        : unwrap(await api.PUT("/admin/exams/{id}", { params: { path: { id } }, body })),
    onSuccess: (saved) => {
      queryClient.setQueryData(examQueryKey(saved.id), saved);
      return queryClient.invalidateQueries({ queryKey: examsQueryKey });
    },
  });
}

export function useRegenerateExamLink(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () =>
      unwrap(await api.POST("/admin/exams/{id}/exam-link", { params: { path: { id } } })),
    // The response is the Exam with the new token, so the page shows the new link without a second request.
    onSuccess: (saved) => queryClient.setQueryData(examQueryKey(saved.id), saved),
  });
}

export function useDeleteExam() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) =>
      unwrapEmpty(await api.DELETE("/admin/exams/{id}", { params: { path: { id } } })),
    onSuccess: (_result, id) => {
      queryClient.removeQueries({ queryKey: examQueryKey(id) });
      return queryClient.invalidateQueries({ queryKey: examsQueryKey });
    },
  });
}
