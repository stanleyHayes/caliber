import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { authApi } from '../api/auth';
import type { LoginInput, RegisterInput } from '../api/types';
import { useAuthStore } from '../stores/auth';

export function useMe() {
  const accessToken = useAuthStore((s) => s.accessToken);
  return useQuery({
    queryKey: ['me'],
    queryFn: async () => {
      const { user } = await authApi.me();
      useAuthStore.getState().setUser(user);
      return user;
    },
    enabled: Boolean(accessToken),
  });
}

export function useLogin() {
  return useMutation({
    mutationFn: (input: LoginInput) => authApi.login(input),
    onSuccess: (data) =>
      useAuthStore.getState().setSession(data.user, data.tokens.accessToken, data.tokens.refreshToken),
  });
}

export function useRegister() {
  return useMutation({
    mutationFn: (input: RegisterInput) => authApi.register(input),
    onSuccess: (data) =>
      useAuthStore.getState().setSession(data.user, data.tokens.accessToken, data.tokens.refreshToken),
  });
}

export function useLogout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const rt = useAuthStore.getState().refreshToken;
      if (rt) {
        await authApi.logout(rt);
      }
    },
    onSettled: () => {
      useAuthStore.getState().clear();
      qc.clear();
    },
  });
}
