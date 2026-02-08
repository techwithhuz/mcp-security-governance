import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'MCP Governance Dashboard',
  description: 'Enterprise MCP Security Governance & Compliance Dashboard',
  icons: {
    icon: '/favicon.svg',
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="dark">
      <body className="min-h-screen bg-gov-bg">
        {children}
      </body>
    </html>
  );
}
