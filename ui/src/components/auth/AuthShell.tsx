type AuthShellProps = {
  title: string;
  description: string;
  children: React.ReactNode;
  footer: React.ReactNode;
};

export default function AuthShell({
  title,
  description,
  children,
  footer,
}: AuthShellProps) {
  return (
    <div className="flex min-h-screen w-full flex-1 flex-col overflow-y-auto lg:w-1/2">
      <div className="mx-auto flex w-full max-w-md flex-1 items-start px-6 py-10 sm:px-10 lg:px-8 lg:py-14">
        <div className="auth-panel-enter w-full">
          <div className="mb-8">
            <h1 className="mb-2 text-3xl font-semibold text-gray-900 dark:text-white sm:text-4xl">
              {title}
            </h1>
            <p className="text-sm leading-6 text-gray-500 dark:text-gray-400">
              {description}
            </p>
          </div>

          <div className="space-y-6">{children}</div>

          <div className="mt-6 text-sm text-gray-600 dark:text-gray-400">{footer}</div>
        </div>
      </div>
    </div>
  );
}
