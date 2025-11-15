import './globals.css'

export const metadata = {
  title: 'Pablo Card Game',
  description: 'Play Pablo with your friends!',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}

