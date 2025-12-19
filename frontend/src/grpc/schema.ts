import { proto3, ScalarType, MethodKind, Empty, FieldMask, Timestamp } from "@bufbuild/protobuf";

export const Song = proto3.makeMessageType(
  "song.Song",
  () => [
    { no: 1, name: "id", kind: "scalar", T: ScalarType.UINT64 },
    { no: 2, name: "title", kind: "scalar", T: ScalarType.STRING },
    { no: 3, name: "description", kind: "scalar", T: ScalarType.STRING },
    { no: 4, name: "link", kind: "scalar", T: ScalarType.STRING }
  ]
);

export type SongType = InstanceType<typeof Song>;

export const ListSongsRequest = proto3.makeMessageType(
  "song.ListSongsRequest",
  () => [
    { no: 1, name: "parent", kind: "scalar", T: ScalarType.STRING },
    { no: 2, name: "page_size", kind: "scalar", T: ScalarType.INT32 },
    { no: 3, name: "page_token", kind: "scalar", T: ScalarType.STRING }
  ]
);

export const ListSongsResponse = proto3.makeMessageType(
  "song.ListSongsResponse",
  () => [
    { no: 1, name: "songs", kind: "message", T: Song, repeated: true },
    { no: 2, name: "next_page_token", kind: "scalar", T: ScalarType.STRING }
  ]
);

export const GetSongRequest = proto3.makeMessageType(
  "song.GetSongRequest",
  () => [{ no: 1, name: "name", kind: "scalar", T: ScalarType.STRING }]
);

export const CreateSongRequest = proto3.makeMessageType(
  "song.CreateSongRequest",
  () => [
    { no: 1, name: "parent", kind: "scalar", T: ScalarType.STRING },
    { no: 2, name: "song_id", kind: "scalar", T: ScalarType.STRING },
    { no: 3, name: "song", kind: "message", T: Song }
  ]
);

export const UpdateSongRequest = proto3.makeMessageType(
  "song.UpdateSongRequest",
  () => [
    { no: 1, name: "song", kind: "message", T: Song },
    {
      no: 2,
      name: "update_mask",
      kind: "message",
      T: FieldMask
    }
  ]
);

export const DeleteSongRequest = proto3.makeMessageType(
  "song.DeleteSongRequest",
  () => [{ no: 1, name: "name", kind: "scalar", T: ScalarType.STRING }]
);

export const SongService = {
  typeName: "song.SongService",
  methods: {
    listSongs: {
      name: "ListSongs",
      I: ListSongsRequest,
      O: ListSongsResponse,
      kind: MethodKind.Unary
    },
    getSong: {
      name: "GetSong",
      I: GetSongRequest,
      O: Song,
      kind: MethodKind.Unary
    },
    createSong: {
      name: "CreateSong",
      I: CreateSongRequest,
      O: Song,
      kind: MethodKind.Unary
    },
    updateSong: {
      name: "UpdateSong",
      I: UpdateSongRequest,
      O: Song,
      kind: MethodKind.Unary
    },
    deleteSong: {
      name: "DeleteSong",
      I: DeleteSongRequest,
      O: Empty,
      kind: MethodKind.Unary
    }
  }
} as const;

export const Concert = proto3.makeMessageType(
  "concert.Concert",
  () => [
    { no: 1, name: "id", kind: "scalar", T: ScalarType.UINT64 },
    { no: 2, name: "name", kind: "scalar", T: ScalarType.STRING },
    { no: 3, name: "date", kind: "message", T: Timestamp }
  ]
);

export type ConcertType = InstanceType<typeof Concert>;

export const ListConcertsRequest = proto3.makeMessageType(
  "concert.ListConcertsRequest",
  () => [
    { no: 1, name: "parent", kind: "scalar", T: ScalarType.STRING },
    { no: 2, name: "page_size", kind: "scalar", T: ScalarType.INT32 },
    { no: 3, name: "page_token", kind: "scalar", T: ScalarType.STRING }
  ]
);

export const ListConcertsResponse = proto3.makeMessageType(
  "concert.ListConcertsResponse",
  () => [
    { no: 1, name: "concerts", kind: "message", T: Concert, repeated: true },
    { no: 2, name: "next_page_token", kind: "scalar", T: ScalarType.STRING }
  ]
);

export const GetConcertRequest = proto3.makeMessageType(
  "concert.GetConcertRequest",
  () => [{ no: 1, name: "name", kind: "scalar", T: ScalarType.STRING }]
);

export const CreateConcertRequest = proto3.makeMessageType(
  "concert.CreateConcertRequest",
  () => [
    { no: 1, name: "parent", kind: "scalar", T: ScalarType.STRING },
    { no: 2, name: "concert_id", kind: "scalar", T: ScalarType.STRING },
    { no: 3, name: "concert", kind: "message", T: Concert }
  ]
);

export const UpdateConcertRequest = proto3.makeMessageType(
  "concert.UpdateConcertRequest",
  () => [
    { no: 1, name: "concert", kind: "message", T: Concert },
    {
      no: 2,
      name: "update_mask",
      kind: "message",
      T: FieldMask
    }
  ]
);

export const DeleteConcertRequest = proto3.makeMessageType(
  "concert.DeleteConcertRequest",
  () => [{ no: 1, name: "name", kind: "scalar", T: ScalarType.STRING }]
);

export const ConcertService = {
  typeName: "concert.ConcertService",
  methods: {
    listConcerts: {
      name: "ListConcerts",
      I: ListConcertsRequest,
      O: ListConcertsResponse,
      kind: MethodKind.Unary
    },
    getConcert: {
      name: "GetConcert",
      I: GetConcertRequest,
      O: Concert,
      kind: MethodKind.Unary
    },
    createConcert: {
      name: "CreateConcert",
      I: CreateConcertRequest,
      O: Concert,
      kind: MethodKind.Unary
    },
    updateConcert: {
      name: "UpdateConcert",
      I: UpdateConcertRequest,
      O: Concert,
      kind: MethodKind.Unary
    },
    deleteConcert: {
      name: "DeleteConcert",
      I: DeleteConcertRequest,
      O: Empty,
      kind: MethodKind.Unary
    }
  }
} as const;

export const Participation = proto3.makeMessageType(
  "participation.Participation",
  () => [
    { no: 1, name: "tg_id", kind: "scalar", T: ScalarType.UINT64 },
    { no: 2, name: "song_id", kind: "scalar", T: ScalarType.UINT64 },
    { no: 3, name: "role_title", kind: "scalar", T: ScalarType.STRING }
  ]
);

export type ParticipationType = InstanceType<typeof Participation>;
export type TgLoginType = InstanceType<typeof TgLogin>;

export const ListParticipationsRequest = proto3.makeMessageType(
  "participation.ListParticipationsRequest",
  () => [
    { no: 1, name: "parent", kind: "scalar", T: ScalarType.STRING },
    { no: 2, name: "page_size", kind: "scalar", T: ScalarType.INT32 },
    { no: 3, name: "page_token", kind: "scalar", T: ScalarType.STRING }
  ]
);

export const ListParticipationsResponse = proto3.makeMessageType(
  "participation.ListParticipationsResponse",
  () => [
    {
      no: 1,
      name: "participations",
      kind: "message",
      T: Participation,
      repeated: true
    },
    { no: 2, name: "next_page_token", kind: "scalar", T: ScalarType.STRING }
  ]
);

export const GetParticipationRequest = proto3.makeMessageType(
  "participation.GetParticipationRequest",
  () => [{ no: 1, name: "name", kind: "scalar", T: ScalarType.STRING }]
);

export const CreateParticipationRequest = proto3.makeMessageType(
  "participation.CreateParticipationRequest",
  () => [
    { no: 1, name: "parent", kind: "scalar", T: ScalarType.STRING },
    { no: 2, name: "participation_id", kind: "scalar", T: ScalarType.STRING },
    { no: 3, name: "participation", kind: "message", T: Participation }
  ]
);

export const UpdateParticipationRequest = proto3.makeMessageType(
  "participation.UpdateParticipationRequest",
  () => [
    { no: 1, name: "participation", kind: "message", T: Participation },
    {
      no: 2,
      name: "update_mask",
      kind: "message",
      T: FieldMask
    }
  ]
);

export const DeleteParticipationRequest = proto3.makeMessageType(
  "participation.DeleteParticipationRequest",
  () => [{ no: 1, name: "name", kind: "scalar", T: ScalarType.STRING }]
);

export const ParticipationService = {
  typeName: "participation.ParticipationService",
  methods: {
    listParticipations: {
      name: "ListParticipations",
      I: ListParticipationsRequest,
      O: ListParticipationsResponse,
      kind: MethodKind.Unary
    },
    getParticipation: {
      name: "GetParticipation",
      I: GetParticipationRequest,
      O: Participation,
      kind: MethodKind.Unary
    },
    createParticipation: {
      name: "CreateParticipation",
      I: CreateParticipationRequest,
      O: Participation,
      kind: MethodKind.Unary
    },
    updateParticipation: {
      name: "UpdateParticipation",
      I: UpdateParticipationRequest,
      O: Participation,
      kind: MethodKind.Unary
    },
    deleteParticipation: {
      name: "DeleteParticipation",
      I: DeleteParticipationRequest,
      O: Empty,
      kind: MethodKind.Unary
    }
  }
} as const;

export const TgLogin = proto3.makeMessageType(
  "auth.TgLogin",
  () => [{ no: 1, name: "tg_id", kind: "scalar", T: ScalarType.UINT64 }]
);

/*
message LoginResponse {
	// JWT token
	string token = 1;
	// Is specified user an admin
	bool is_admin = 2;
	// Time of creation of the token
	uint64 iat = 3;
	// Time of expiration of the token
	uint64 exp = 4;
}
*/
export const LoginResponse = proto3.makeMessageType(
  "auth.LoginResponse",
  () => [
    { no: 1, name: "token", kind: "scalar", T: ScalarType.STRING },
    { no: 2, name: "is_admin", kind: "scalar", T: ScalarType.BOOL },
    { no: 3, name: "iat", kind: "scalar", T: ScalarType.UINT64 },
    { no: 4, name: "exp", kind: "scalar", T: ScalarType.UINT64 }
  ]
);

export const AuthService = {
  typeName: "auth.AuthService",
  methods: {
    loginTg: {
      name: "LoginTg",
      I: TgLogin,
      O: LoginResponse,
      kind: MethodKind.Unary
    }
  }
} as const;
