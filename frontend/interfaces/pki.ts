export type IPkiTemplate = {
  profile_id: number;
  name: string;
  common_name: string;
  san: string;
  seat_id: string;
};

export type IPkiConfig = { name: string; templates: IPkiTemplate[] };
