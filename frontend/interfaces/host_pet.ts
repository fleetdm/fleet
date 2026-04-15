export type HostPetSpecies = "cat";

export type HostPetMood =
  | "happy"
  | "content"
  | "sad"
  | "hungry"
  | "dirty"
  | "sick";

export type HostPetAction = "feed" | "play" | "clean" | "medicine";

export interface IHostPet {
  id: number;
  host_id: number;
  name: string;
  species: HostPetSpecies;
  health: number;
  happiness: number;
  hunger: number;
  cleanliness: number;
  mood: HostPetMood;
  last_interacted_at: string;
  created_at: string;
  updated_at: string;
}
