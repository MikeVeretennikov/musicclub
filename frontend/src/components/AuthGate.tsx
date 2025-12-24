import React, { useState, useEffect } from "react";
import { createPortal } from "react-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";

import { getProfile, getTgLoginLink, login, logout, register, telegramWebAppAuth } from "../services/api";
import { setTokenPair } from "../services/config";
import { CredentialsSchema, RegisterUserRequestSchema } from "../proto/auth_pb";
import SongList from "./SongList";
import EventList from "./EventList";
import type { PermissionSet } from "../proto/permissions_pb";
import { UserSchema, type User } from "../proto/user_pb";
import { create } from "@bufbuild/protobuf";

const AuthGate: React.FC = () => {
	const queryClient = useQueryClient();
	const [authError, setAuthError] = useState<string | null>(null);
	const [isAuthenticating, setIsAuthenticating] = useState(false);
	const [tgLinkError, setTgLinkError] = useState<string | null>(null);

	const profileQuery = useQuery({
		queryKey: ["profile"],
		queryFn: () => getProfile(),
		retry: false,
	});

	const isUnauthedCode = profileQuery.isError && (profileQuery.error as ConnectError)?.code === Code.Unauthenticated;
	const profile = profileQuery.data?.profile as User | undefined;
	const permissions = profileQuery.data?.permissions as PermissionSet | undefined;
	const [isProfileOpen, setProfileOpen] = useState(false);
	const tgLoginLinkMutation = useMutation({
		mutationFn: () => getTgLoginLink(profile ? { id: profile.id } : undefined),
	});

	// Auto-authenticate via Telegram WebApp
	useEffect(() => {
		const tg = window.Telegram?.WebApp;

		console.log("🔧 [DEBUG] Telegram WebApp:", tg);
		console.log("🔧 [DEBUG] initData:", tg?.initData);
		console.log("🔧 [DEBUG] initDataUnsafe:", tg?.initDataUnsafe);

		if (!tg || !tg.initData) {
			console.warn("⚠️ Not running in Telegram WebApp or initData is empty");
			return;
		}

		// Signal to Telegram that the app is ready
		tg.ready();
		tg.expand();

		const performTelegramAuth = async () => {
			if (isAuthenticating || profileQuery.data) {
				return;
			}

			setIsAuthenticating(true);
			setAuthError(null);

			console.log("🔐 Authenticating with initData:", tg.initData);

			try {
				const session = await telegramWebAppAuth(tg.initData);

				if (session.tokens?.accessToken == null || session.tokens?.refreshToken == null) {
					setAuthError("Сервер не вернул токены авторизации");
					setIsAuthenticating(false);
					return;
				}

				setTokenPair(session.tokens.accessToken, session.tokens.refreshToken);
				await queryClient.invalidateQueries({ queryKey: ["profile"] });
			} catch (err: any) {
				if (err instanceof ConnectError) {
					setAuthError(err.message);
				} else {
					setAuthError((err as Error).message);
				}
			} finally {
				setIsAuthenticating(false);
			}
		};

		if (isUnauthedCode && !isAuthenticating) {
			performTelegramAuth();
		}
	}, [isUnauthedCode, isAuthenticating, profileQuery.data, queryClient]);

	if (profileQuery.isLoading) {
		return (
			<div className="card" style={{ maxWidth: 400, margin: "80px auto" }}>
				<div className="card-title">Загружаем профиль…</div>
				<div style={{ textAlign: "center", padding: "40px 0" }}>
					<div className="spinner" />
				</div>
			</div>
		);
	}

	if (isUnauthedCode) {
		// Check if we're in Telegram WebApp
		const tg = window.Telegram?.WebApp;

		if (!tg || !tg.initData) {
			// Not in Telegram - show error message
			return (
				<div className="card" style={{ maxWidth: 400, margin: "80px auto" }}>
					<div className="card-title" style={{ marginBottom: 16 }}>
						<span role="img" aria-label="music">
							🎸
						</span>
						Музыкальный клуб
					</div>
					<p style={{ color: "var(--muted)", lineHeight: 1.4, marginBottom: 24 }}>
						Это приложение доступно только через Telegram Mini App.
					</p>
					<div style={{
						padding: "16px",
						backgroundColor: "var(--accent-bg)",
						border: "1px solid var(--accent)",
						borderRadius: "8px",
						color: "var(--text)"
					}}>
						<strong style={{ display: "block", marginBottom: 8 }}>Как открыть приложение:</strong>
						<ol style={{ margin: 0, paddingLeft: 20, lineHeight: 1.6 }}>
							<li>Откройте Telegram</li>
							<li>Найдите бота @{window.location.hostname.includes('localhost') ? 'mikeaiogrambot' : 'YourBotUsername'}</li>
							<li>Нажмите кнопку "Открыть приложение"</li>
						</ol>
					</div>
				</div>
			);
		}

		// In Telegram, show authenticating state
		return (
			<div className="card" style={{ maxWidth: 400, margin: "80px auto" }}>
				<div className="card-title" style={{ marginBottom: 16 }}>
					<span role="img" aria-label="music">
						🎸
					</span>
					Музыкальный клуб
				</div>
				{authError ? (
					<div style={{
						padding: "16px",
						backgroundColor: "var(--danger-bg)",
						border: "1px solid var(--danger)",
						borderRadius: "8px",
						color: "var(--danger)",
						marginBottom: 16
					}}>
						{authError}
					</div>
				) : (
					<div style={{ textAlign: "center", padding: "40px 0" }}>
						<div className="spinner" style={{ marginBottom: 16 }} />
						<p style={{ color: "var(--muted)" }}>
							Авторизация через Telegram...
						</p>
					</div>
				)}
			</div>
		);
	}

	if (profileQuery.isError) {
		return (
			<div className="card" style={{ maxWidth: 400, margin: "80px auto" }}>
				<div className="card-title">Ошибка</div>
				<div style={{ padding: "20px", textAlign: "center" }}>
					<p style={{ color: "var(--danger)", marginBottom: 16 }}>
						Ошибка загрузки профиля: {(profileQuery.error as Error).message}
					</p>
					<button
						className="button"
						onClick={() => profileQuery.refetch()}
					>
						Попробовать снова
					</button>
				</div>
			</div>
		);
	}

	const hero = (
		<div className="card" style={{ marginBottom: 18 }}>
			<div className="section-header">
				<div className="card-title">
					<span role="img" aria-label="music">
						🎸
					</span>
					Музыкальный клуб
				</div>
				<button
					type="button"
					className="pill"
					style={{ cursor: "pointer" }}
					onClick={() => setProfileOpen(true)}
				>
					{profile?.avatarUrl ? (
						<img
							src={profile.avatarUrl}
							alt={profile.displayName}
							className="avatar-small"
						/>
					) : (
						<div
							className="status-dot"
							style={{ background: profile ? "var(--accent)" : "var(--muted)" }}
						/>
					)}
					{profile?.displayName}
				</button>
			</div>
			<p style={{ color: "var(--muted)", marginBottom: 12 }}>
				Собираем сет-листы, треклисты и роли для ближайших мероприятий.
			</p>
			{profileQuery.data && "isChatMember" in profileQuery.data ? null : null}
			{profile && !profile.telegramId && (
				<div className="pill" style={{ justifyContent: "space-between", alignItems: "center", gap: 12 }}>
					<div style={{ flex: 1, minWidth: 0 }}>
						<div style={{ fontWeight: 600, marginBottom: 4 }}>Привяжите Telegram</div>
						<small style={{ color: "var(--muted)" }}>Получите ссылку для авторизации в боте</small>
						{tgLinkError && (
							<div style={{ color: "var(--danger)", marginTop: 8 }}>{tgLinkError}</div>
						)}
					</div>
					<button
						className="button"
						type="button"
						disabled={tgLoginLinkMutation.isPending}
						onClick={async () => {
							setTgLinkError(null);
							try {
								const res = await tgLoginLinkMutation.mutateAsync();
								if (res.loginLink) {
									window.open(res.loginLink, "_blank", "noopener");
								}
							} catch (err: any) {
								if (err instanceof ConnectError) {
									setTgLinkError(err.message);
								} else {
									setTgLinkError((err as Error).message);
								}
							}
						}}
					>
						{tgLoginLinkMutation.isPending ? "Получаем..." : "Получить ссылку"}
					</button>
				</div>
			)}
		</div>
	);

	return (
		<div className="grid">
			{hero}
			<SongList permissions={permissions} profile={profile} />
			<EventList permissions={permissions} />
			{isProfileOpen && profile && (
				<ProfileModal profile={profile} onClose={() => setProfileOpen(false)} />
			)}
		</div>
	);
};

const ProfileModal: React.FC<{ profile: User; onClose: () => void }> = ({ profile, onClose }) => {
	return createPortal(
		<div className="modal-backdrop" onClick={onClose}>
			<div className="card modal-window" onClick={(e) => e.stopPropagation()}>
				<div className="section-header">
					<div className="card-title">
						<span role="img" aria-label="user">
							👤
						</span>
						{profile.displayName}
					</div>
					<button className="button secondary" onClick={onClose}>
						Закрыть
					</button>
				</div>
				<div style={{ color: "var(--muted)", marginBottom: 12 }}>
					Профиль пользователя
				</div>
				<div className="grid">
					<div className="pill" style={{ justifyContent: "space-between" }}>
						<span>Имя пользователя</span>
						<strong>{profile.username}</strong>
					</div>
					{profile.avatarUrl && (
						<div className="pill" style={{ display: "flex", justifyContent: "space-between", gap: 12, alignItems: "center" }}>
							<span>Аватар</span>
							<img
								src={profile.avatarUrl}
								alt={profile.displayName}
								className="avatar-small"
							/>
						</div>
					)}
				</div>
				<div style={{ marginTop: 18, display: "flex", gap: 10, justifyContent: "flex-end" }}>
					<button className="button danger" onClick={() => logout()}>
						Выйти
					</button>
				</div>
			</div>
		</div>,
		document.body,
	);
};

export default AuthGate;
