import { NextRequest, NextResponse } from 'next/server';

/**
 * Runtime proxy middleware — rewrites /api/* requests to the governance
 * controller. CONTROLLER_API_URL is read at request time so it works
 * correctly in standalone mode regardless of build-time env.
 */
export function middleware(request: NextRequest) {
  const controllerUrl =
    process.env.CONTROLLER_API_URL || 'http://localhost:8090';

  const { pathname, search } = request.nextUrl;
  const destination = `${controllerUrl}${pathname}${search}`;

  return NextResponse.rewrite(new URL(destination));
}

export const config = {
  matcher: '/api/:path*',
};
