import "./globals.css";

export const metadata = {
  title: "Orfeu Web",
  description: "Orfeu no navegador, com motor Go/WebAssembly e autenticação no servidor de simulação.",
  icons: {
    icon: "/favicon.ico",
    shortcut: "/favicon.ico",
  },
};

export default function RootLayout({ children }) {
  return (
    <html lang="pt-BR">
      <body>{children}</body>
    </html>
  );
}
