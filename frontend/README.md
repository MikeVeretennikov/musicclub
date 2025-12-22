# Music Club Frontend

React + Vite UI tailored for Telegram Mini App embedding. It talks to the gRPC backend defined in `api/proto`.

## Run locally

```bash
cd frontend
npm install
npm run dev
```

## Generating proto

```bash
npx buf generate
```

## Notes

- Auth uses `auth.AuthService/LoginTg` with a Telegram ID and stores the JWT in `localStorage`, attaching it to gRPC calls as a bearer token.
- Admin-only actions (create/delete) are toggled based on `auth.User/IsAdmin`.
- Uses `@bufbuild/connect-web` with handwritten service definitions that mirror the protobuf files, so no codegen step is required.
