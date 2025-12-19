import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  concertClient,
  participationClient,
  songClient,
  authClient,
} from "./client";
import {
  type ConcertType,
  type ParticipationType,
  type SongType,
  type TgLoginType
} from "./schema";
import { PlainMessage } from "@bufbuild/protobuf";
import { saveToken, clearToken } from "../lib/auth";

const defaultListParams = { parent: "", page_size: 100, page_token: "" };

export function useSongs() {
  return useQuery({
    queryKey: ["songs"],
    queryFn: async () => {
      const res = await songClient.listSongs(defaultListParams);
      return res.songs;
    }
  });
}

export function useConcerts() {
  return useQuery({
    queryKey: ["concerts"],
    queryFn: async () => {
      const res = await concertClient.listConcerts(defaultListParams);
      return res.concerts;
    }
  });
}

export function useParticipations() {
  return useQuery({
    queryKey: ["participations"],
    queryFn: async () => {
      const res = await participationClient.listParticipations(
        defaultListParams
      );
      return res.participations;
    }
  });
}

export function useCreateSong() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (song: PlainMessage<SongType>) => {
      const res = await songClient.createSong({
        parent: "",
        song_id: `${song.title}-${Date.now()}`,
        song
      });
      return res;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["songs"] })
  });
}

export function useCreateConcert() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (concert: PlainMessage<ConcertType>) => {
      const res = await concertClient.createConcert({
        parent: "",
        concert_id: `${concert.name}-${Date.now()}`,
        concert
      });
      return res;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["concerts"] })
  });
}

export function useCreateParticipation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (participation: PlainMessage<ParticipationType>) => {
      const res = await participationClient.createParticipation({
        parent: "",
        participation_id: `${participation.tg_id}-${participation.song_id}`,
        participation
      });
      return res;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["participations"] })
  });
}

export function useDeleteSong() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number | string) => {
      await songClient.deleteSong({ name: String(id) });
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["songs"] })
  });
}

export function useDeleteConcert() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number | string) => {
      await concertClient.deleteConcert({ name: String(id) });
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["concerts"] })
  });
}

export function useDeleteParticipation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (compositeKey: string) => {
      await participationClient.deleteParticipation({ name: compositeKey });
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["participations"] })
  });
}

export function useLogin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (tgId: string) => {
      const payload: PlainMessage<TgLoginType> = {
        tg_id: BigInt(tgId)
      };
      console.log("Logging in with payload:", payload);
      const res = await authClient.loginTg(payload);
      console.log("Server response:", res);
      saveToken(res.token);
      return { token: res.token, isAdmin: res.is_admin };
    },
    onSuccess: () => qc.invalidateQueries()
  });
}

export function useLogout() {
  const qc = useQueryClient();
  return () => {
    clearToken();
    qc.clear();
  };
}
