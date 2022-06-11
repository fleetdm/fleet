import PropTypes from "prop-types";

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

interface ITableColumn {
  description: string;
  name: string;
  type: string;
  hidden: boolean;
  required: boolean;
  index: boolean;
}

export interface IOsqueryTable {
  columns: ITableColumn[];
  description: string;
  name: string;
  platform?: string;
  url: string;
  platforms: string[];
  evented: boolean;
  cacheable: boolean;
}

export const DEFAULT_OSQUERY_TABLE: IOsqueryTable = {
  name: "users",
  description:
    "Local user accounts (including domain accounts that have logged on locally (Windows)).",
  url: "https://github.com/osquery/osquery/blob/master/specs/users.table",
  platforms: ["darwin", "linux", "windows", "freebsd"],
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
    },
    {
      name: "uid_signed",
      description: "User ID as int64 signed (Apple)",
      type: "bigint",
      hidden: false,
      required: false,
      index: false,
    },
    {
      name: "gid_signed",
      description: "Default group ID as int64 signed (Apple)",
      type: "bigint",
      hidden: false,
      required: false,
      index: false,
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
    },
    {
      name: "directory",
      description: "User's home directory",
      type: "text",
      hidden: false,
      required: false,
      index: false,
    },
    {
      name: "shell",
      description: "User's configured default shell",
      type: "text",
      hidden: false,
      required: false,
      index: false,
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
    },
    {
      name: "is_hidden",
      description: "IsHidden attribute set in OpenDirectory",
      type: "integer",
      hidden: false,
      required: false,
      index: false,
    },
    {
      name: "pid_with_namespace",
      description: "Pids that contain a namespace",
      type: "integer",
      hidden: true,
      required: false,
      index: false,
    },
  ],
};
