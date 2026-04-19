export type PasswordChecklist = {
  minLength: boolean;
  lowercase: boolean;
  uppercase: boolean;
  number: boolean;
  special: boolean;
};

export function getPasswordChecklist(password: string): PasswordChecklist {
  return {
    minLength: password.length >= 8,
    lowercase: /[a-z]/.test(password),
    uppercase: /[A-Z]/.test(password),
    number: /[0-9]/.test(password),
    special: /[^A-Za-z0-9]/.test(password),
  };
}

export function isStrongPassword(password: string): boolean {
  const checklist = getPasswordChecklist(password);
  return Object.values(checklist).every(Boolean);
}

export async function parseAPIError(response: Response): Promise<string> {
  try {
    const data = (await response.json()) as {
      message?: string;
      error?: string;
    };
    if (typeof data.message === "string" && data.message.trim() !== "") {
      return data.message;
    }
    if (typeof data.error === "string" && data.error.trim() !== "") {
      return data.error;
    }
  } catch {
    return "Something went wrong. Please try again.";
  }

  return "Something went wrong. Please try again.";
}

export function isForgotPasswordTokenError(error: string): boolean {
  const normalized = error.toLowerCase();
  return (
    normalized.includes("forgot password token") ||
    normalized.includes("reset token") ||
    normalized.includes("reset session")
  );
}
