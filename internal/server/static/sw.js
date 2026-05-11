const CACHE = 'chore-v1';
const STATIC = ['/static/app.css', '/static/htmx.min.js', '/static/manifest.json'];

self.addEventListener('install', e => {
  e.waitUntil(caches.open(CACHE).then(c => c.addAll(STATIC)));
  self.skipWaiting();
});

self.addEventListener('activate', e => {
  e.waitUntil(
    caches.keys().then(keys =>
      Promise.all(keys.filter(k => k !== CACHE).map(k => caches.delete(k)))
    )
  );
  self.clients.claim();
});

self.addEventListener('fetch', e => {
  const url = new URL(e.request.url);

  // Cache-first for static assets
  if (url.pathname.startsWith('/static/') || url.pathname === '/manifest.json') {
    e.respondWith(
      caches.match(e.request).then(cached => cached || fetch(e.request).then(res => {
        const clone = res.clone();
        caches.open(CACHE).then(c => c.put(e.request, clone));
        return res;
      }))
    );
    return;
  }

  // Network-first for HTML / htmx fragments — fall back to a simple offline page
  e.respondWith(
    fetch(e.request).catch(() =>
      new Response('<p style="font-family:sans-serif;padding:2rem">Offline — connect to your NAS to use Chore Scheduler.</p>',
        { headers: { 'Content-Type': 'text/html' } })
    )
  );
});
