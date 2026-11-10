export type Department = "techdept" | "clinic" | "digimedia" | "techsvc";

export interface Position {
  code: string;
  label: string;
  scope: "association" | "department";
  department: Department | null;
}

export interface Permissions {
  systemAdmin: boolean;
  manager: boolean;
  accessDepartments: Department[];
  managedDepartments: Department[];
}

export interface UserSummary {
  subject: string;
  userName: string;
  avatar: string | null;
  positions: Position[];
}

export interface Profile extends UserSummary {
  email: string | null;
  permissions: Permissions | null;
}

export interface AuthStatus {
  state: "anonymous" | "active";
  user: Profile | null;
}
