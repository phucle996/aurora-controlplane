import './globals.css';
import "flatpickr/dist/flatpickr.css";
import AuthBootstrap from '@/components/auth/AuthBootstrap';
import { ToastProvider } from '@/components/ui/toast/ToastProvider';
import { SidebarProvider } from '@/context/SidebarContext';
import { ThemeProvider } from '@/context/ThemeContext';

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="dark:bg-gray-900">
        <ThemeProvider>
          <ToastProvider>
            <SidebarProvider>
              <AuthBootstrap />
              {children}
            </SidebarProvider>
          </ToastProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
