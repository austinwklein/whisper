<script>
  import { onMount } from 'svelte';
  import { Login, Register, GetPeerInfo, GetMultiaddr, IsLoggedIn, GetCurrentUser } from '../wailsjs/go/main/App';

  let username = '';
  let password = '';
  let fullName = '';
  let isLoggedIn = false;
  let isRegistering = false;
  let peerInfo = '';
  let multiaddr = '';
  let currentUser = null;
  let error = '';
  let loading = false;

  onMount(async () => {
    // Check if already logged in
    try {
      isLoggedIn = await IsLoggedIn();
      if (isLoggedIn) {
        await loadUserData();
      }
    } catch (e) {
      console.error('Failed to check login status:', e);
      error = e.toString();
    }
  });

  async function loadUserData() {
    try {
      currentUser = await GetCurrentUser();
      peerInfo = await GetPeerInfo();
      multiaddr = await GetMultiaddr();
      username = currentUser?.username || '';
    } catch (e) {
      console.error('Failed to load user data:', e);
    }
  }

  async function handleLogin() {
    if (!username || !password) {
      error = 'Username and password are required';
      return;
    }

    loading = true;
    error = '';

    try {
      await Login(username, password);
      isLoggedIn = true;
      await loadUserData();
      password = '';
    } catch (e) {
      error = e.toString() || 'Login failed';
      console.error('Login error:', e);
    } finally {
      loading = false;
    }
  }

  async function handleRegister() {
    if (!username || !password || !fullName) {
      error = 'All fields are required';
      return;
    }

    if (password.length < 8) {
      error = 'Password must be at least 8 characters';
      return;
    }

    loading = true;
    error = '';

    try {
      await Register(username, password, fullName);
      // Auto-login after registration
      await Login(username, password);
      isLoggedIn = true;
      await loadUserData();
      password = '';
      isRegistering = false;
    } catch (e) {
      error = e.toString() || 'Registration failed';
      console.error('Registration error:', e);
    } finally {
      loading = false;
    }
  }

  function toggleMode() {
    isRegistering = !isRegistering;
    error = '';
    password = '';
  }
</script>

<main>
  <div class="container">
    <h1>🤫 Whisper</h1>
    <p class="subtitle">Decentralized P2P Chat</p>

    {#if !isLoggedIn}
      <div class="auth-section">
        <h2>{isRegistering ? 'Register' : 'Login'}</h2>

        {#if error}
          <div class="error">{error}</div>
        {/if}

        <input
          type="text"
          bind:value={username}
          placeholder="Username"
          class="input"
          disabled={loading}
        />

        <input
          type="password"
          bind:value={password}
          placeholder="Password"
          class="input"
          disabled={loading}
        />

        {#if isRegistering}
          <input
            type="text"
            bind:value={fullName}
            placeholder="Full Name"
            class="input"
            disabled={loading}
          />
        {/if}

        <button
          on:click={isRegistering ? handleRegister : handleLogin}
          class="btn-primary"
          disabled={loading}
        >
          {loading ? 'Please wait...' : (isRegistering ? 'Register' : 'Login')}
        </button>

        <button on:click={toggleMode} class="btn-secondary" disabled={loading}>
          {isRegistering ? 'Already have an account? Login' : 'Need an account? Register'}
        </button>
      </div>
    {:else}
      <div class="chat-section">
        <h2>Welcome, {currentUser?.fullName || username}!</h2>

        <div class="info-card">
          <h3>Your Peer Info</h3>
          <div class="peer-info">
            <strong>Peer ID:</strong>
            <code>{peerInfo || 'Loading...'}</code>
          </div>
          <div class="peer-info">
            <strong>Multiaddress:</strong>
            <code>{multiaddr || 'Loading...'}</code>
          </div>
        </div>

        <div class="status">
          <span class="status-indicator online"></span>
          <span>Connected to P2P Network</span>
        </div>

        <div class="placeholder">
          <p>🚧 More features coming soon!</p>
          <ul>
            <li>Friend Management</li>
            <li>Direct Messaging</li>
            <li>Conference Chats</li>
          </ul>
        </div>
      </div>
    {/if}
  </div>
</main>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen,
      Ubuntu, Cantarell, sans-serif;
    background: #1a1a1a;
    color: #ffffff;
  }

  main {
    min-height: 100vh;
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 20px;
  }

  .container {
    max-width: 600px;
    width: 100%;
    background: #2a2a2a;
    padding: 40px;
    border-radius: 12px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  }

  h1 {
    margin: 0 0 10px 0;
    font-size: 3em;
    text-align: center;
  }

  h2 {
    margin: 0 0 20px 0;
    text-align: center;
    color: #e0e0e0;
  }

  h3 {
    margin: 0 0 15px 0;
    font-size: 1.2em;
    color: #b0b0b0;
  }

  .subtitle {
    text-align: center;
    color: #888;
    margin: 0 0 30px 0;
  }

  .auth-section {
    display: flex;
    flex-direction: column;
    gap: 15px;
  }

  .error {
    padding: 12px;
    background: #dc2626;
    color: white;
    border-radius: 8px;
    font-size: 14px;
    text-align: center;
  }

  .input {
    padding: 12px 16px;
    border: 2px solid #444;
    border-radius: 8px;
    background: #1a1a1a;
    color: #fff;
    font-size: 16px;
  }

  .input:focus {
    outline: none;
    border-color: #6366f1;
  }

  .input:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary {
    padding: 12px 24px;
    background: #6366f1;
    color: white;
    border: none;
    border-radius: 8px;
    font-size: 16px;
    cursor: pointer;
    transition: background 0.2s;
  }

  .btn-primary:hover:not(:disabled) {
    background: #4f46e5;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-secondary {
    padding: 8px 16px;
    background: transparent;
    color: #6366f1;
    border: 1px solid #6366f1;
    border-radius: 8px;
    font-size: 14px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-secondary:hover:not(:disabled) {
    background: #6366f1;
    color: white;
  }

  .btn-secondary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .chat-section {
    text-align: center;
  }

  .info-card {
    margin: 20px 0;
    padding: 20px;
    background: #1a1a1a;
    border-radius: 8px;
    text-align: left;
  }

  .peer-info {
    margin: 10px 0;
    padding: 10px;
    background: #2a2a2a;
    border-radius: 6px;
    font-size: 14px;
  }

  .peer-info strong {
    display: block;
    margin-bottom: 5px;
    color: #888;
    font-size: 12px;
    text-transform: uppercase;
  }

  .peer-info code {
    display: block;
    font-family: 'Monaco', 'Courier New', monospace;
    font-size: 12px;
    color: #6366f1;
    word-break: break-all;
    line-height: 1.4;
  }

  .status {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    margin: 20px 0;
  }

  .status-indicator {
    width: 12px;
    height: 12px;
    border-radius: 50%;
  }

  .status-indicator.online {
    background: #10b981;
    box-shadow: 0 0 8px #10b981;
  }

  .placeholder {
    margin-top: 30px;
    padding: 20px;
    background: #1a1a1a;
    border-radius: 8px;
    border: 2px dashed #444;
  }

  .placeholder p {
    margin: 0 0 15px 0;
    font-size: 18px;
  }

  .placeholder ul {
    list-style: none;
    padding: 0;
    margin: 0;
    text-align: left;
  }

  .placeholder li {
    padding: 8px 0;
    color: #888;
  }

  .placeholder li::before {
    content: '→ ';
    color: #6366f1;
    font-weight: bold;
  }
</style>
