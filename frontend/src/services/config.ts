import { type Interceptor } from "@connectrpc/connect";
import { createGrpcWebTransport } from "@connectrpc/connect-web";

export const BACKEND_URL = import.meta.env.VITE_GRPC_HOST ?? "http://localhost:6969";

let TOKEN = "PLACEHOLDER";

const authInterceptor: Interceptor = (next) => async (req) => {
	if (TOKEN) {
		req.header.set("authorization", `Bearer ${TOKEN}`);
	}
	return await next(req);
};

export const transport = createGrpcWebTransport({
	baseUrl: BACKEND_URL,
	interceptors: [authInterceptor],
});

export function getToken(): string {
	return TOKEN;
}

export function setToken(token: string) {
	TOKEN = token;
}
