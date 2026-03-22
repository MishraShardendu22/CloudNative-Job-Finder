import { cookies } from "next/headers";

const TOKEN_COOKIE_NAME = "jwt";

export async function getServerToken() {
  const cookieStore = await cookies();
  return cookieStore.get(TOKEN_COOKIE_NAME)?.value;
}

export { TOKEN_COOKIE_NAME };
