/** @type {import('next').NextConfig} */
const nextConfig = {
  typescript: {
    ignoreBuildErrors: true,
  },
  images: {
    unoptimized: true,
  },
  // 开发环境API代理配置
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: process.env.NEXT_PUBLIC_API_BASE 
          ? `${process.env.NEXT_PUBLIC_API_BASE}/:path*`
          : 'http://localhost:8080/api/:path*',
      },
    ]
  },
}

export default nextConfig
