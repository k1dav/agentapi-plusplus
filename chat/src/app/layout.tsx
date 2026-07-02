import type { Metadata } from "next";
import { Montserrat, Bricolage_Grotesque, JetBrains_Mono } from "next/font/google";
import "./globals.css";
import { Toaster } from "@/components/ui/sonner";
import { ThemeProvider } from "@/components/theme-provider";

/**
 * Phenotype typography stack — wired via next/font for zero-CLS, self-hosted
 * delivery, and font-feature-settings support. The CSS variables are consumed
 * by `tokens.css` (`--font-display`, `--font-heading`, `--font-body`,
 * `--font-mono`) so both Tailwind utilities and hand-written CSS share the
 * same source of truth.
 */
const montserrat = Montserrat({
  variable: "--font-heading",
  subsets: ["latin"],
  display: "swap",
});

const bricolageGrotesque = Bricolage_Grotesque({
  variable: "--font-display",
  subsets: ["latin"],
  display: "swap",
});

const jetbrainsMono = JetBrains_Mono({
  variable: "--font-mono",
  subsets: ["latin"],
  display: "swap",
});

export const metadata: Metadata = {
  title: "AgentAPI Chat",
  description: "A ChatGPT-like interface for AgentAPI",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body
        className={`${montserrat.variable} ${bricolageGrotesque.variable} ${jetbrainsMono.variable} antialiased`}
      >
        <ThemeProvider
          attribute="class"
          defaultTheme="system"
          enableSystem
          disableTransitionOnChange
        >
          {children}
          <Toaster richColors closeButton />
        </ThemeProvider>
      </body>
    </html>
  );
}