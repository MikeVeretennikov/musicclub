import { Code, ConnectError, type Interceptor, createClient } from "@connectrpc/connect";
import { createGrpcWebTransport } from "@connectrpc/connect-web";

import { AuthService } from "../proto/auth_pb";

export const BACKEND_URL = import.meta.env.VITE_GRPC_HOST ?? "http://localhost:6969";

const ACCESS_TOKEN_COOKIE = "mc_access_token";
const REFRESH_TOKEN_COOKIE = "mc_refresh_token";
const ACCESS_TOKEN_DEFAULT_MAX_AGE = 15 * 60; // 15 minutes
const REFRESH_TOKEN_MAX_AGE = 7 * 24 * 60 * 60; // 7 days
const REFRESH_SKEW_MS = 60_000; // refresh 1 minute before expiry

const refreshTransport = createGrpcWebTransport({ baseUrl: BACKEND_URL });
const refreshClient = createClient(AuthService, refreshTransport);

let accessToken = readCookie(ACCESS_TOKEN_COOKIE) ?? "";
let refreshToken = readCookie(REFRESH_TOKEN_COOKIE) ?? "";
let accessTokenExpiryMs = decodeJwtExpiry(accessToken);
let refreshPromise: Promise<void> | null = null;

function readCookie(name: string): string | undefined {
	const match = document.cookie.match(new RegExp(`(?:^|; )${name}=([^;]*)`));
	if (match != null && match[1] != null) {
		return decodeURIComponent(match[1]);
	} else {
		return undefined;
	}
}

function setCookie(name: string, value: string, maxAgeSeconds: number) {
	const secure = window.location.protocol === "https:" ? "; secure" : "";
	document.cookie = `${name}=${encodeURIComponent(value)}; path=/; max-age=${maxAgeSeconds}; samesite=lax${secure}`;
}

function deleteCookie(name: string) {
	document.cookie = `${name}=; path=/; max-age=0; samesite=lax`;
}

function decodeJwtExpiry(token: string): number | null {
	if (!token) return null;
	try {
		const [, payload] = token.split(".");
		if (!payload) return null;
		const decoded = JSON.parse(atob(payload.replace(/-/g, "+").replace(/_/g, "/")));
		return typeof decoded.exp === "number" ? decoded.exp * 1000 : null;
	} catch {
		return null;
	}
}

function calcAccessMaxAge(expiryMs: number | null): number {
	if (!expiryMs) return ACCESS_TOKEN_DEFAULT_MAX_AGE;
	const deltaSeconds = Math.max(1, Math.floor((expiryMs - Date.now()) / 1000));
	return Math.min(deltaSeconds, ACCESS_TOKEN_DEFAULT_MAX_AGE);
}

export function setTokenPair(newAccessToken?: string, newRefreshToken?: string) {
	accessToken = newAccessToken ?? "";
	refreshToken = newRefreshToken ?? "";
	accessTokenExpiryMs = decodeJwtExpiry(accessToken);

	if (accessToken) {
		setCookie(ACCESS_TOKEN_COOKIE, accessToken, calcAccessMaxAge(accessTokenExpiryMs));
	} else {
		deleteCookie(ACCESS_TOKEN_COOKIE);
	}

	if (refreshToken) {
		setCookie(REFRESH_TOKEN_COOKIE, refreshToken, REFRESH_TOKEN_MAX_AGE);
	} else {
		deleteCookie(REFRESH_TOKEN_COOKIE);
	}
}

export function clearTokenPair() {
	setTokenPair("", "");
}

export function getAccessToken(): string {
	return accessToken;
}

export function getRefreshToken(): string {
	return refreshToken;
}

function shouldRefreshAccess(): boolean {
	if (!accessToken) return Boolean(refreshToken);
	if (accessTokenExpiryMs == null) return false;
	return accessTokenExpiryMs - Date.now() <= REFRESH_SKEW_MS;
}

async function refreshTokens(force = false) {
	if (!refreshToken) {
		if (force) clearTokenPair();
		return;
	}
	if (!force && !shouldRefreshAccess()) return;
	if (refreshPromise) return await refreshPromise;

	refreshPromise = (async () => {
		try {
			const tokenPair = await refreshClient.refresh({ refreshToken });
			if (!tokenPair.accessToken || !tokenPair.refreshToken) {
				throw new Error("missing token pair in refresh response");
			}
			setTokenPair(tokenPair.accessToken, tokenPair.refreshToken);
		} catch (err) {
			clearTokenPair();
			throw err;
		} finally {
			refreshPromise = null;
		}
	})();

	return await refreshPromise;
}

const authInterceptor: Interceptor = (next) => async (req) => {
	let retried = false;

	const prepareAuth = async (forceRefresh = false) => {
		await refreshTokens(forceRefresh);
		if (accessToken) {
			req.header.set("authorization", `Bearer ${accessToken}`);
		}
	};

	await prepareAuth(false);

	try {
		return await next(req);
	} catch (err) {
		if (!retried && err instanceof ConnectError && err.code === Code.Unauthenticated) {
			retried = true;
			await prepareAuth(true);
			if (!accessToken) {
				throw err;
			}
			return await next(req);
		}
		throw err;
	}
};

export const transport = createGrpcWebTransport({
	baseUrl: BACKEND_URL,
	interceptors: [authInterceptor],
});
