import Button from "@/components/ui/button/Button";
import Link from "next/link";
import AuthShell from "./AuthShell";

export default function InvalidResetTokenView() {
  return (
    <AuthShell
      title="Reset link invalid"
      description="Token sai hoặc đã hết hạn."
      footer={
        <Link
          href="/signin"
          className="inline-flex items-center gap-2 text-sm text-gray-600 transition-colors hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
        >
          <span aria-hidden="true">&larr;</span>
          <span>Back to sign in</span>
        </Link>
      }
    >
      <div className="space-y-4">
        <p className="text-sm leading-6 text-gray-500 dark:text-gray-400">
          The reset token is no longer valid. Please start the forgot-password
          flow again to request a new reset email.
        </p>
        <Link href="/signin" className="block">
          <Button className="w-full rounded-lg py-3.5" size="sm">
            Back to sign in
          </Button>
        </Link>
      </div>
    </AuthShell>
  );
}
