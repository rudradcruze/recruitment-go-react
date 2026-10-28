import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "../../api/client";
import { unwrap, unwrapEmpty } from "../../api/errors";
import type { components } from "../../api/schema";

export type LoginRequest = components["schemas"]["LoginRequest"];

// sessionQueryKey is the key of the session of the Admin in the cache of the queries. It starts with the name of the context.
export const sessionQueryKey = ["admin", "session"] as const;

export function useSession() {
  return useQuery({
    queryKey: sessionQueryKey,
    queryFn: async () => unwrap(await api.GET("/admin/session")),
  });
}

export function useLogin() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: LoginRequest) =>
      unwrapEmpty(await api.POST("/admin/session", { body })),
    // The login gives no body. Remove the old session result, so the Admin page reads the new session.
    onSuccess: () => queryClient.removeQueries({ queryKey: sessionQueryKey }),
  });
}

export function useLogout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => unwrapEmpty(await api.DELETE("/admin/session")),
    // Remove all data of the Admin from the cache, so the next Admin on this browser does not see it.
    onSuccess: () => queryClient.clear(),
  });
}
