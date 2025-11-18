<script lang="ts">
  import { onMount } from 'svelte';

  let username = '';
  let isLoggedIn = false;
  let peerInfo = '';

  onMount(() => {
    // Will connect to Wails backend when integrated
    console.log('Whisper GUI loaded');
  });

  function handleLogin() {
    // Placeholder for login functionality
    isLoggedIn = true;
  }
</script>

<main>
  <div class="container">
    <h1>🤫 Whisper</h1>
    <p class="subtitle">Decentralized P2P Chat</p>

    {#if !isLoggedIn}
      <div class="login-section">
        <h2>Login</h2>
        <input
          type="text"
          bind:value={username}
          placeholder="Username"
          class="input"
        />
        <button on:click={handleLogin} class="btn-primary">
          Login
        </button>
      </div>
    {:else}
      <div class="chat-section">
        <h2>Welcome, {username}!</h2>
        <div class="peer-info">
          <p>Peer ID: {peerInfo || 'Loading...'}</p>
        </div>
        <div class="status">
          <span class="status-indicator online"></span>
          <span>Connected</span>
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

  .subtitle {
    text-align: center;
    color: #888;
    margin: 0 0 30px 0;
  }

  .login-section {
    display: flex;
    flex-direction: column;
    gap: 15px;
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

  .btn-primary:hover {
    background: #4f46e5;
  }

  .chat-section {
    text-align: center;
  }

  .peer-info {
    margin: 20px 0;
    padding: 15px;
    background: #1a1a1a;
    border-radius: 8px;
    font-family: monospace;
    font-size: 14px;
    word-break: break-all;
  }

  .status {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    margin-top: 20px;
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
</style>
