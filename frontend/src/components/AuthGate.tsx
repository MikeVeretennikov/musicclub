import React, { useState, useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";

import { getProfile, login, register } from "../services/api";
import { setTokenPair } from "../services/config";
import { CredentialsSchema, RegisterUserRequestSchema } from "../proto/auth_pb";
import SongList from "./SongList";
import EventList from "./EventList";
import type { PermissionSet } from "../proto/permissions_pb";
import { UserSchema, type User } from "../proto/user_pb";
import { create } from "@bufbuild/protobuf";

type AuthMode = "login" | "register";

const AuthGate: React.FC = () => {
	const queryClient = useQueryClient();
	const [authMode, setAuthMode] = useState<AuthMode>("login");
	const [username, setUsername] = useState("");
	const [password, setPassword] = useState("");
	const [displayName, setDisplayName] = useState("");
	const [avatarUrl, setAvatarUrl] = useState("");
	const [authError, setAuthError] = useState<string | null>(null);
	const [isLoading, setIsLoading] = useState(false);

	const profileQuery = useQuery({
		queryKey: ["profile"],
		queryFn: () => getProfile(),
		retry: false, // Don't retry on auth errors
	});

	const isUnauthedCode = profileQuery.isError && (profileQuery.error as ConnectError)?.code === Code.Unauthenticated;
	const profile = profileQuery.data?.profile as User | undefined;
	const permissions = profileQuery.data?.permissions as PermissionSet | undefined;

	const handleAuthSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		setAuthError(null);
		setIsLoading(true);

		try {
			let session;

			if (authMode === "register") {
				session = await register(create(RegisterUserRequestSchema, {
					credentials: create(CredentialsSchema, { username, password }),
					profile: create(UserSchema, {
						displayName: displayName || username,
						avatarUrl: avatarUrl || "",
					}),
				}));
			} else {
				session = await login(create(CredentialsSchema, { username, password }));
			}

			if (session.tokens?.accessToken == null || session.tokens?.refreshToken == null) {
				setAuthError("server didn't return token pair")
				setIsLoading(false);
				return
			}

			setTokenPair(session.tokens?.accessToken, session.tokens?.refreshToken);
			await queryClient.invalidateQueries({ queryKey: ["profile"] });
		} catch (err: any) {
			if (err instanceof ConnectError) {
				setAuthError(err.message);
			} else {
				setAuthError((err as Error).message);
			}
		} finally {
			setIsLoading(false);
		}
	};

	// Clear form when switching modes
	useEffect(() => {
		setAuthError(null);
		if (authMode === "login") {
			setDisplayName("");
			setAvatarUrl("");
		}
	}, [authMode]);

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
		return (
			<div className="card" style={{ maxWidth: 400, margin: "80px auto" }}>
				<div className="card-title" style={{ marginBottom: 16 }}>
					<span role="img" aria-label="music">
						🎸
					</span>
					Музыкальный клуб
				</div>
				<p style={{ color: "var(--muted)", lineHeight: 1.4, marginBottom: 24 }}>
					{authMode === "login"
						? "Войдите в свой аккаунт для доступа к клубу"
						: "Создайте новый аккаунт для участия в клубе"}
				</p>

				<form onSubmit={handleAuthSubmit}>
					<div style={{ display: "grid", gap: 16 }}>
						<div>
							<label className="form-label">Имя пользователя</label>
							<input
								className="input"
								type="text"
								placeholder="username"
								value={username}
								onChange={(e) => setUsername(e.target.value)}
								required
								disabled={isLoading}
							/>
						</div>

						<div>
							<label className="form-label">Пароль</label>
							<input
								className="input"
								type="password"
								placeholder="password"
								value={password}
								onChange={(e) => setPassword(e.target.value)}
								required
								disabled={isLoading}
								minLength={8}
							/>
							{authMode === "login" && (
								<small style={{ color: "var(--muted)", display: "block", marginTop: 4 }}>
									Минимум 8 символов
								</small>
							)}
						</div>

						{authMode === "register" && (
							<>
								<div>
									<label className="form-label">Отображаемое имя (необязательно)</label>
									<input
										className="input"
										type="text"
										placeholder="Иван Иванов"
										value={displayName}
										onChange={(e) => setDisplayName(e.target.value)}
										disabled={isLoading}
									/>
									<small style={{ color: "var(--muted)", display: "block", marginTop: 4 }}>
										Если не указано, будет использовано имя пользователя
									</small>
								</div>

								<div>
									<label className="form-label">Аватар URL (необязательно)</label>
									<input
										className="input"
										type="url"
										placeholder="https://example.com/avatar.jpg"
										value={avatarUrl}
										onChange={(e) => setAvatarUrl(e.target.value)}
										disabled={isLoading}
									/>
								</div>
							</>
						)}

						{authError && (
							<div style={{
								padding: "12px",
								backgroundColor: "var(--danger-bg)",
								border: "1px solid var(--danger)",
								borderRadius: "8px",
								color: "var(--danger)"
							}}>
								{authError}
							</div>
						)}

						<button
							className="button"
							type="submit"
							disabled={isLoading || !username || !password}
							style={{ width: "100%" }}
						>
							{isLoading ? (
								<>
									<span className="spinner" style={{ marginRight: 8 }} />
									{authMode === "login" ? "Входим..." : "Регистрируем..."}
								</>
							) : (
								authMode === "login" ? "Войти" : "Зарегистрироваться"
							)}
						</button>

						<div style={{ textAlign: "center", marginTop: 8 }}>
							<button
								type="button"
								className="button secondary"
								onClick={() => setAuthMode(authMode === "login" ? "register" : "login")}
								disabled={isLoading}
							>
								{authMode === "login"
									? "Нет аккаунта? Зарегистрироваться"
									: "Уже есть аккаунт? Войти"}
							</button>
						</div>
					</div>
				</form>
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
				<div className="pill">
					<div
						className="status-dot"
						style={{ background: profile ? "var(--accent)" : "var(--muted)" }}
					/>
					{profile?.displayName}
				</div>
			</div>
			<p style={{ color: "var(--muted)", marginBottom: 12 }}>
				Собираем сет-листы, треклисты и роли для ближайших мероприятий.
			</p>
			{profileQuery.data && "isChatMember" in profileQuery.data ? null : null}
		</div>
	);

	return (
		<div className="grid">
			{hero}
			<SongList permissions={permissions} profile={profile} />
			<EventList permissions={permissions} />
		</div>
	);
};

export default AuthGate;