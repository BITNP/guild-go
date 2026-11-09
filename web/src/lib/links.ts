export function safeReturnTo(value: unknown): string {
  if (typeof value !== "string") {
    return "/";
  }

  if (!value.startsWith("/") || value.startsWith("//")) {
    return "/";
  }

  if (/[\\\s\p{Cc}]/u.test(value)) {
    return "/";
  }

  try {
    const url = new URL(value, "https://guild.invalid");

    if (url.origin !== "https://guild.invalid") {
      return "/";
    }

    if (url.pathname === "/login" || url.pathname.startsWith("/api/")) {
      return "/";
    }

    return `${url.pathname}${url.search}${url.hash}`;
  } catch {
    return "/";
  }
}
