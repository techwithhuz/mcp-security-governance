import { NextRequest, NextResponse } from 'next/server';

/**
 * Generic proxy route for all /api/governance/* endpoints.
 * Forwards requests to the MCP Governance Controller API.
 * 
 * Usage:
 * - GET /api/governance/score → GET {CONTROLLER_API}/api/governance/score
 * - GET /api/governance/findings → GET {CONTROLLER_API}/api/governance/findings
 * etc.
 */

const CONTROLLER_API_URL = process.env.CONTROLLER_API_URL || 'http://localhost:8090';

export async function GET(
  request: NextRequest,
  { params }: { params: { path: string[] } }
) {
  const path = params.path.join('/');
  const url = new URL(request.url);
  const searchParams = url.searchParams.toString();
  
  const controllerUrl = `${CONTROLLER_API_URL}/api/governance/${path}${
    searchParams ? `?${searchParams}` : ''
  }`;

  try {
    const response = await fetch(controllerUrl, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      cache: 'no-store',
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: `Controller API returned ${response.status}` },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error(`[governance-proxy] Failed to fetch ${controllerUrl}:`, error);
    return NextResponse.json(
      { 
        error: 'Failed to connect to governance controller',
        details: error instanceof Error ? error.message : 'Unknown error',
      },
      { status: 503 }
    );
  }
}

export async function POST(
  request: NextRequest,
  { params }: { params: { path: string[] } }
) {
  const path = params.path.join('/');
  
  const controllerUrl = `${CONTROLLER_API_URL}/api/governance/${path}`;

  try {
    const body = await request.json().catch(() => ({}));
    
    const response = await fetch(controllerUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
      cache: 'no-store',
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: `Controller API returned ${response.status}` },
        { status: response.status }
      );
    }

    const data = await response.json();
    return NextResponse.json(data);
  } catch (error) {
    console.error(`[governance-proxy] Failed to POST to ${controllerUrl}:`, error);
    return NextResponse.json(
      { 
        error: 'Failed to connect to governance controller',
        details: error instanceof Error ? error.message : 'Unknown error',
      },
      { status: 503 }
    );
  }
}
