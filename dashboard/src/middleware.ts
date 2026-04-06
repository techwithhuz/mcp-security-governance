import { NextRequest, NextResponse } from 'next/server';

/**
 * Runtime proxy middleware — rewrites /api/* requests to the governance
 * controller. CONTROLLER_API_URL is read at request time so it works
 * correctly in standalone mode regardless of build-time env.
 * 
 * EXCLUDES: /api/scan/* (local endpoint for repository scanning)
 */
export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Skip rewrite for local scan endpoints (handled by Next.js route handlers)
  if (pathname.startsWith('/api/scan/') || pathname.startsWith('/api/governance/scan/')) {
    return NextResponse.next();
  }


  const controllerUrl =
    process.env.CONTROLLER_API_URL || 'http://localhost:8090';

  const { search } = request.nextUrl;
  const destination = `${controllerUrl}${pathname}${search}`;

  return NextResponse.rewrite(new URL(destination));
}

export const config = {
  matcher: '/api/:path*',
};
