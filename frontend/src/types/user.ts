export type User = {
  id: string;
  email: string;
  name?: string;
  created_at?: string;
};

export type AuthPayload = {
  email: string;
  password: string;
};
