import { createClient, Interceptor } from "@connectrpc/connect";
import { createGrpcWebTransport } from "@connectrpc/connect-web";

import {
  AuthService,
  ConcertService,
  ParticipationService,
  SongService,
} from "./schema";
import { getToken } from "../lib/auth";

const baseUrl = import.meta.env.VITE_GRPC_HOST ?? "http://backend:6969";

const authInterceptor: Interceptor = (next) => async (req) => {
  const token = getToken();
  if (token) {
    req.header.set("Authorization", `Bearer ${token}`);
  }
  return next(req);
};

const transport = createGrpcWebTransport({
  baseUrl,
  useBinaryFormat: true,
  interceptors: [authInterceptor]
});

export const songClient = createClient(SongService, transport);
export const concertClient = createClient(ConcertService, transport);
export const participationClient = createClient(
  ParticipationService,
  transport
);
export const authClient = createClient(AuthService, transport);
