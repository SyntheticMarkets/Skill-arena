const nextConfig = {
  reactStrictMode: true,
  output: process.env.NEXT_OUTPUT === 'standalone' ? 'standalone' : undefined,
}

export default nextConfig
