import PropTypes from "prop-types";
import { QueryablePlatform, QueryableDisplayPlatform } from "./platform";

export default PropTypes.shape({
  columns: PropTypes.arrayOf(
    PropTypes.shape({
      description: PropTypes.string,
      name: PropTypes.string,
      type: PropTypes.string,
    })
  ),
  description: PropTypes.string,
  name: PropTypes.string,
  platform: PropTypes.string,
});

export type ColumnType =
  | "integer"
  | "bigint"
  | "double"
  | "text"
  | "unsigned_bigint"
  | "STRING"
  | "string"; // TODO: Why do we have type string, STRING, and text in schema.json?

// TODO: Replace with one or the other once osquery_fleet_schema.json follows one type or other
export type TableSchemaPlatform = QueryableDisplayPlatform | QueryablePlatform;
export interface IQueryTableColumn {
  name: string;
  description: string;
  type: ColumnType;
  hidden: boolean;
  required: boolean;
  index: boolean;
  platforms?: TableSchemaPlatform[];
  requires_user_context?: boolean;
}

export interface IOsQueryTable {
  name: string;
  description: string;
  url: string;
  platforms: TableSchemaPlatform[];
  evented: boolean;
  cacheable: boolean;
  columns: IQueryTableColumn[];
  examples?: string;
  notes?: string;
  hidden?: boolean;
}

// Also used for testing
export const DEFAULT_OSQUERY_TABLE: IOsQueryTable = {
  name: "users",
  description:
    "Local user accounts (including domain accounts that have logged on locally (Windows)).",
  url: "https://github.com/osquery/osquery/blob/master/specs/users.table",
  platforms: ["darwin", "linux", "windows", "chrome"],
  evented: false,
  cacheable: false,
  columns: [
    {
      name: "uid",
      description: "User ID",
      type: "bigint",
      hidden: false,
      required: false,
      index: false,
    },
    {
      name: "gid",
      description: "Group ID (unsigned)",
      type: "bigint",
      hidden: false,
      required: false,
      index: false,
      platforms: ["macOS", "Windows", "Linux"],
    },
    {
      name: "uid_signed",
      description: "User ID as int64 signed (Apple)",
      type: "bigint",
      hidden: false,
      required: false,
      index: false,
      platforms: ["macOS", "Windows", "Linux"],
    },
    {
      name: "gid_signed",
      description: "Default group ID as int64 signed (Apple)",
      type: "bigint",
      hidden: false,
      required: false,
      index: false,
      platforms: ["macOS", "Windows", "Linux"],
    },
    {
      name: "username",
      description: "Username",
      type: "text",
      hidden: false,
      required: false,
      index: false,
    },
    {
      name: "description",
      description: "Optional user description",
      type: "text",
      hidden: false,
      required: false,
      index: false,
      platforms: ["macOS", "Windows", "Linux"],
    },
    {
      name: "directory",
      description: "User's home directory",
      type: "text",
      hidden: false,
      required: false,
      index: false,
      platforms: ["macOS", "Windows", "Linux"],
    },
    {
      name: "shell",
      description: "User's configured default shell",
      type: "text",
      hidden: false,
      required: false,
      index: false,
      platforms: ["macOS", "Windows", "Linux"],
    },
    {
      name: "uuid",
      description: "User's UUID (Apple) or SID (Windows)",
      type: "text",
      hidden: false,
      required: false,
      index: false,
    },
    {
      name: "type",
      description:
        "Whether the account is roaming (domain), local, or a system profile",
      type: "text",
      hidden: true,
      required: false,
      index: false,
      platforms: ["Windows"],
    },
    {
      name: "is_hidden",
      description: "IsHidden attribute set in OpenDirectory",
      type: "integer",
      hidden: false,
      required: false,
      index: false,
      platforms: ["macOS"],
    },
    {
      name: "pid_with_namespace",
      description: "Pids that contain a namespace",
      type: "integer",
      hidden: true,
      required: false,
      index: false,
    },
    {
      name: "email",
      description: "Email",
      type: "text",
      hidden: false,
      required: false,
      index: false,
      platforms: ["chrome"],
    },
  ],
  notes: "",
  examples:
    "List users that have interactive access via a shell that isn't false.\n```\nSELECT * FROM users WHERE shell!='/usr/bin/false';\n```",
};
