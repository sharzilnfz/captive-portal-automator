import http from 'http';

export class MockPortal {
  constructor(port = 8080) {
    this.port = port;
    this.isOnline = false;
    this.server = null;
    this.lastRequest = null;
  }

  start() {
    return new Promise((resolve, reject) => {
      this.server = http.createServer((req, res) => {
        this.lastRequest = {
          url: req.url,
          method: req.method,
          headers: req.headers
        };

        // 1. Simulate Apple's Connectivity Check
        if (req.url === '/hotspot-detect.html') {
          if (this.isOnline) {
            res.writeHead(200, { 'Content-Type': 'text/html' });
            res.end('<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>');
          } else {
            // Redirect to captive portal landing page using relative URL
            res.writeHead(302, { 'Location': '/login' });
            res.end();
          }
          return;
        }

        // 2. Serve the Captive Portal Login Page
        if (req.url === '/login' && req.method === 'GET') {
          res.writeHead(200, { 'Content-Type': 'text/html' });
          res.end(`
            <html>
              <body>
                <form action="/auth" method="POST">
                  <input type="hidden" name="csrf" value="mock-csrf-token-123">
                  <input type="hidden" name="session" value="xyz987">
                  <label>Username: <input type="text" name="username_field"></label>
                  <label>Password: <input type="password" name="password_field"></label>
                  <input type="submit" value="Login">
                </form>
              </body>
            </html>
          `);
          return;
        }

        // 3. Handle Authentication Submit
        if (req.url.startsWith('/auth')) {
          if (req.method === 'POST') {
            let body = '';
            req.on('data', chunk => { body += chunk; });
            req.on('end', () => {
              const params = new URLSearchParams(body);
              if (
                params.get('username_field') === 'student123' &&
                params.get('password_field') === 'securepass' &&
                params.get('csrf') === 'mock-csrf-token-123'
              ) {
                this.isOnline = true;
                res.writeHead(200, { 'Content-Type': 'text/html' });
                res.end('<h1>Logged In Successfully!</h1>');
              } else {
                res.writeHead(401, { 'Content-Type': 'text/html' });
                res.end('<h1>Unauthorized</h1>');
              }
            });
          } else if (req.method === 'GET') {
            const urlObj = new URL(req.url, `http://localhost:${this.port}`);
            const params = urlObj.searchParams;
            if (
              params.get('username_field') === 'student123' &&
              params.get('password_field') === 'securepass' &&
              params.get('csrf') === 'mock-csrf-token-123'
            ) {
              this.isOnline = true;
              res.writeHead(200, { 'Content-Type': 'text/html' });
              res.end('<h1>Logged In Successfully!</h1>');
            } else {
              res.writeHead(401, { 'Content-Type': 'text/html' });
              res.end('<h1>Unauthorized</h1>');
            }
          }
          return;
        }

        // 4. Default Not Found
        res.writeHead(404);
        res.end();
      });

      const onError = (err) => {
        cleanup();
        reject(err);
      };

      const onListening = () => {
        cleanup();
        resolve();
      };

      const cleanup = () => {
        if (this.server) {
          this.server.removeListener('error', onError);
          this.server.removeListener('listening', onListening);
        }
      };

      this.server.once('error', onError);
      this.server.once('listening', onListening);
      this.server.listen(this.port);
    });
  }

  stop() {
    return new Promise((resolve) => {
      if (this.server) {
        this.server.close(() => resolve());
      } else {
        resolve();
      }
    });
  }
}
