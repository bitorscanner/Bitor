import type { RecordModel } from "pocketbase";

export type Settings = RecordModel & {
  collectionId: string;
  favicon: string;
  setup_completed: boolean;
  website_title: string;
};
