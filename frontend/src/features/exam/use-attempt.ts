import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "../../api/client";
import { unwrap, unwrapEmpty } from "../../api/errors";
import type { components } from "../../api/schema";

export type StartAttemptRequest = components["schemas"]["StartAttemptRequest"];
export type Attempt = components["schemas"]["Attempt"];
export type Score = components["schemas"]["Score"];

// The keys of the queries start with the name of the context.
export const examLinkQueryKey = (token: string) => ["exam", "exam-link", token] as const;
export const attemptQueryKey = (id: string) => ["exam", "attempt", id] as const;

export function useExamLink(token: string) {
  return useQuery({
    queryKey: examLinkQueryKey(token),
    queryFn: async () =>
      unwrap(await api.GET("/exam-links/{token}", { params: { path: { token } } })),
  });
}

export function useStartAttempt(token: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: StartAttemptRequest) =>
      unwrap(await api.POST("/exam-links/{token}/attempts", { params: { path: { token } }, body })),
    // The response has the Attempt, so the Attempt page does not depend on a second request.
    onSuccess: (started) => queryClient.setQueryData(attemptQueryKey(started.id), started),
  });
}

export function useAttempt(id: string) {
  return useQuery({
    queryKey: attemptQueryKey(id),
    queryFn: async () => unwrap(await api.GET("/attempts/{id}", { params: { path: { id } } })),
    // The page keeps the Answers in its own state. A refresh in the background must not replace them,
    // and it must not show a timed-out Attempt in place of the Score that the Submit gives.
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  });
}

export function useSaveAnswer(attemptId: string) {
  return useMutation({
    mutationFn: async ({ questionId, optionId }: { questionId: string; optionId: string }) =>
      unwrapEmpty(
        await api.PUT("/attempts/{id}/answers/{questionId}", {
          params: { path: { id: attemptId, questionId } },
          body: { optionId },
        }),
      ),
  });
}

export function useSubmitAttempt(attemptId: string) {
  return useMutation({
    mutationFn: async () =>
      unwrap(await api.POST("/attempts/{id}/submission", { params: { path: { id: attemptId } } })),
  });
}
