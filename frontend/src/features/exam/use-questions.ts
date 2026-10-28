import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "../../api/client";
import { unwrap, unwrapEmpty } from "../../api/errors";
import type { components } from "../../api/schema";

export type Question = components["schemas"]["Question"];
export type QuestionRequest = components["schemas"]["QuestionRequest"];

// The keys of the queries start with the name of the context.
export const questionsQueryKey = ["exam", "questions"] as const;
export const questionQueryKey = (id: string) => ["exam", "question", id] as const;

export function useQuestions() {
  return useQuery({
    queryKey: questionsQueryKey,
    queryFn: async () => unwrap(await api.GET("/admin/questions")),
  });
}

export function useQuestion(id: string | undefined) {
  return useQuery({
    queryKey: questionQueryKey(id ?? ""),
    queryFn: async () =>
      unwrap(await api.GET("/admin/questions/{id}", { params: { path: { id: id ?? "" } } })),
    enabled: id !== undefined,
  });
}

// useSaveQuestion creates a Question, or changes the Question with this ID.
export function useSaveQuestion(id: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: QuestionRequest) =>
      id === undefined
        ? unwrap(await api.POST("/admin/questions", { body }))
        : unwrap(await api.PUT("/admin/questions/{id}", { params: { path: { id } }, body })),
    onSuccess: (saved) => {
      queryClient.setQueryData(questionQueryKey(saved.id), saved);
      return queryClient.invalidateQueries({ queryKey: questionsQueryKey });
    },
  });
}

export function useDeleteQuestion() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) =>
      unwrapEmpty(await api.DELETE("/admin/questions/{id}", { params: { path: { id } } })),
    onSuccess: (_result, id) => {
      queryClient.removeQueries({ queryKey: questionQueryKey(id) });
      return queryClient.invalidateQueries({ queryKey: questionsQueryKey });
    },
  });
}
