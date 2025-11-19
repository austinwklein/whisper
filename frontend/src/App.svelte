<script>
  import { onMount } from 'svelte';
  import {
    Login,
    Register,
    Logout,
    GetPeerInfo,
    GetMultiaddr,
    IsLoggedIn,
    GetCurrentUser,
    GetFriends,
    SendFriendRequest,
    GetFriendRequests,
    AcceptFriendRequest,
    RejectFriendRequest,
    SendMessage,
    GetMessages,
    GetUnreadCount
  } from '../wailsjs/go/main/App';

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

  // Friends and messaging state
  let friends = [];
  let friendRequests = [];
  let selectedFriend = null;
  let messages = [];
  let messageContent = '';
  let showAddFriend = false;
  let newFriendMultiaddr = '';
  let newFriendUsername = '';
  let unreadCount = 0;

  // Auto-refresh interval
  let refreshInterval;

  onMount(async () => {
    // Check if already logged in
    try {
      isLoggedIn = await IsLoggedIn();
      if (isLoggedIn) {
        await loadUserData();
        startAutoRefresh();
      }
    } catch (e) {
      console.error('Failed to check login status:', e);
      error = e.toString();
    }
  });

  function startAutoRefresh() {
    // Refresh friends and messages every 3 seconds
    refreshInterval = setInterval(async () => {
      if (isLoggedIn) {
        await refreshFriends();
        await refreshFriendRequests();
        await refreshUnreadCount();
        if (selectedFriend) {
          await loadMessages(selectedFriend.username);
        }
      }
    }, 3000);
  }

  async function loadUserData() {
    try {
      currentUser = await GetCurrentUser();
      peerInfo = await GetPeerInfo();
      multiaddr = await GetMultiaddr();
      username = currentUser?.username || '';
      await refreshFriends();
      await refreshFriendRequests();
      await refreshUnreadCount();
    } catch (e) {
      console.error('Failed to load user data:', e);
    }
  }

  async function refreshFriends() {
    try {
      friends = await GetFriends();
    } catch (e) {
      console.error('Failed to refresh friends:', e);
    }
  }

  async function refreshFriendRequests() {
    try {
      friendRequests = await GetFriendRequests();
      console.log('DEBUG: Friend requests:', friendRequests);
    } catch (e) {
      console.error('Failed to refresh friend requests:', e);
    }
  }

  async function refreshUnreadCount() {
    try {
      unreadCount = await GetUnreadCount();
    } catch (e) {
      console.error('Failed to refresh unread count:', e);
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
      startAutoRefresh();
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
      startAutoRefresh();
      password = '';
      isRegistering = false;
    } catch (e) {
      error = e.toString() || 'Registration failed';
      console.error('Registration error:', e);
    } finally {
      loading = false;
    }
  }

  async function handleLogout() {
    try {
      if (refreshInterval) {
        clearInterval(refreshInterval);
      }
      await Logout();
      isLoggedIn = false;
      currentUser = null;
      friends = [];
      friendRequests = [];
      selectedFriend = null;
      messages = [];
      username = '';
      password = '';
    } catch (e) {
      error = 'Failed to logout: ' + e.toString();
    }
  }

  function toggleMode() {
    isRegistering = !isRegistering;
    error = '';
    password = '';
  }

  function copyToClipboard(text, label) {
    navigator.clipboard.writeText(text).then(() => {
      alert(`${label} copied to clipboard!`);
    }).catch(err => {
      console.error('Failed to copy:', err);
      alert('Failed to copy to clipboard');
    });
  }

  async function handleAddFriend() {
    if (!newFriendMultiaddr || !newFriendUsername) {
      error = 'Both multiaddress and username are required';
      return;
    }

    loading = true;
    error = '';

    try {
      await SendFriendRequest(newFriendMultiaddr, newFriendUsername);
      newFriendMultiaddr = '';
      newFriendUsername = '';
      showAddFriend = false;
      await refreshFriends();
      alert('Friend request sent!');
    } catch (e) {
      error = 'Failed to send friend request: ' + e.toString();
    } finally {
      loading = false;
    }
  }

  async function handleAcceptRequest(username) {
    try {
      await AcceptFriendRequest(username);
      await refreshFriendRequests();
      await refreshFriends();
      alert(`Friend request from ${username} accepted!`);
    } catch (e) {
      error = 'Failed to accept friend request: ' + e.toString();
    }
  }

  async function handleRejectRequest(username) {
    try {
      await RejectFriendRequest(username);
      await refreshFriendRequests();
      alert(`Friend request from ${username} rejected`);
    } catch (e) {
      error = 'Failed to reject friend request: ' + e.toString();
    }
  }

  async function selectFriend(friend) {
    selectedFriend = friend;
    await loadMessages(friend.username);
  }

  async function loadMessages(friendUsername) {
    try {
      messages = await GetMessages(friendUsername, 50);
    } catch (e) {
      console.error('Failed to load messages:', e);
      error = 'Failed to load messages: ' + e.toString();
    }
  }

  async function handleSendMessage() {
    if (!messageContent.trim() || !selectedFriend) {
      return;
    }

    const content = messageContent;
    messageContent = '';

    try {
      await SendMessage(selectedFriend.username, content);
      await loadMessages(selectedFriend.username);
    } catch (e) {
      error = 'Failed to send message: ' + e.toString();
      messageContent = content; // Restore message on error
    }
  }

  function handleKeyPress(event) {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      handleSendMessage();
    }
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
        <div class="header">
          <h2>Welcome, {currentUser?.fullName || username}!</h2>
          <button on:click={handleLogout} class="btn-logout">Logout</button>
        </div>

        {#if error}
          <div class="error">{error}</div>
        {/if}

        <div class="info-card">
          <h3>Your Peer Info</h3>
          <div class="peer-info">
            <div class="info-row">
              <div class="info-content">
                <strong>Peer ID:</strong>
                <code>{peerInfo || 'Loading...'}</code>
              </div>
              <button on:click={() => copyToClipboard(peerInfo, 'Peer ID')} class="btn-copy">
                📋 Copy
              </button>
            </div>
          </div>
          <div class="peer-info">
            <div class="info-row">
              <div class="info-content">
                <strong>Multiaddress:</strong>
                <code>{multiaddr || 'Loading...'}</code>
              </div>
              <button on:click={() => copyToClipboard(multiaddr, 'Multiaddress')} class="btn-copy">
                📋 Copy
              </button>
            </div>
          </div>
        </div>

        <div class="status">
          <span class="status-indicator online"></span>
          <span>Connected to P2P Network</span>
        </div>

        <!-- Friend Requests Section -->
        {#if friendRequests.length > 0}
          <div class="friend-requests">
            <h3>Friend Requests ({friendRequests.length})</h3>
            {#each friendRequests as request}
              <div class="request-item">
                <div class="request-info">
                  <strong>{request.fullName}</strong>
                  <span class="username">@{request.username}</span>
                </div>
                <div class="request-actions">
                  <button on:click={() => handleAcceptRequest(request.username)} class="btn-accept">
                    ✓ Accept
                  </button>
                  <button on:click={() => handleRejectRequest(request.username)} class="btn-reject">
                    ✗ Reject
                  </button>
                </div>
              </div>
            {/each}
          </div>
        {/if}

        <div class="main-content">
          <!-- Friends Sidebar -->
          <div class="friends-sidebar">
            <div class="sidebar-header">
              <h3>Friends ({friends.length})</h3>
              <button on:click={() => showAddFriend = !showAddFriend} class="btn-add">
                + Add
              </button>
            </div>

            {#if showAddFriend}
              <div class="add-friend-form">
                <input
                  type="text"
                  bind:value={newFriendMultiaddr}
                  placeholder="Friend's Multiaddress"
                  class="input-small"
                />
                <input
                  type="text"
                  bind:value={newFriendUsername}
                  placeholder="Friend's Username"
                  class="input-small"
                />
                <div class="form-actions">
                  <button on:click={handleAddFriend} class="btn-submit" disabled={loading}>
                    Send Request
                  </button>
                  <button on:click={() => showAddFriend = false} class="btn-cancel">
                    Cancel
                  </button>
                </div>
              </div>
            {/if}

            <div class="friends-list">
              {#if friends.length === 0}
                <p class="empty-state">No friends yet. Add someone to get started!</p>
              {:else}
                {#each friends as friend}
                  <div
                    class="friend-item {selectedFriend?.username === friend.username ? 'selected' : ''}"
                    on:click={() => selectFriend(friend)}
                  >
                    <span class="status-dot {friend.online ? 'online' : 'offline'}"></span>
                    <div class="friend-info">
                      <strong>{friend.fullName}</strong>
                      <span class="username">@{friend.username}</span>
                    </div>
                  </div>
                {/each}
              {/if}
            </div>
          </div>

          <!-- Chat Area -->
          <div class="chat-area">
            {#if selectedFriend}
              <div class="chat-header">
                <div>
                  <h3>{selectedFriend.fullName}</h3>
                  <span class="chat-status">
                    <span class="status-dot {selectedFriend.online ? 'online' : 'offline'}"></span>
                    {selectedFriend.online ? 'Online' : 'Offline'}
                  </span>
                </div>
              </div>

              <div class="messages-container">
                {#if messages.length === 0}
                  <p class="empty-state">No messages yet. Start the conversation!</p>
                {:else}
                  {#each messages as message}
                    <div class="message {message.fromMe ? 'sent' : 'received'}">
                      <div class="message-content">{message.content}</div>
                      <div class="message-meta">
                        {message.createdAt}
                        {#if message.fromMe}
                          {#if message.read}
                            <span class="status-icon">✓✓</span>
                          {:else if message.delivered}
                            <span class="status-icon">✓</span>
                          {/if}
                        {/if}
                      </div>
                    </div>
                  {/each}
                {/if}
              </div>

              <div class="message-input-container">
                <input
                  type="text"
                  bind:value={messageContent}
                  on:keypress={handleKeyPress}
                  placeholder="Type a message..."
                  class="message-input"
                />
                <button on:click={handleSendMessage} class="btn-send" disabled={!messageContent.trim()}>
                  Send
                </button>
              </div>
            {:else}
              <div class="no-selection">
                <p>Select a friend to start chatting</p>
              </div>
            {/if}
          </div>
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
    max-width: 1200px;
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
    max-width: 400px;
    margin: 0 auto;
  }

  .error {
    padding: 12px;
    background: #dc2626;
    color: white;
    border-radius: 8px;
    font-size: 14px;
    text-align: center;
  }

  .input, .input-small {
    padding: 12px 16px;
    border: 2px solid #444;
    border-radius: 8px;
    background: #1a1a1a;
    color: #fff;
    font-size: 16px;
  }

  .input-small {
    font-size: 14px;
    padding: 8px 12px;
  }

  .input:focus, .input-small:focus {
    outline: none;
    border-color: #6366f1;
  }

  .input:disabled, .input-small:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary, .btn-secondary, .btn-logout, .btn-copy, .btn-add, .btn-accept, .btn-reject, .btn-submit, .btn-cancel, .btn-send {
    padding: 12px 24px;
    border: none;
    border-radius: 8px;
    font-size: 16px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-primary {
    background: #6366f1;
    color: white;
  }

  .btn-primary:hover:not(:disabled) {
    background: #4f46e5;
  }

  .btn-secondary {
    padding: 8px 16px;
    background: transparent;
    color: #6366f1;
    border: 1px solid #6366f1;
    font-size: 14px;
  }

  .btn-secondary:hover:not(:disabled) {
    background: #6366f1;
    color: white;
  }

  .btn-logout {
    padding: 8px 16px;
    background: #dc2626;
    color: white;
    font-size: 14px;
  }

  .btn-logout:hover {
    background: #b91c1c;
  }

  .btn-copy {
    padding: 6px 12px;
    background: #6366f1;
    color: white;
    font-size: 12px;
    white-space: nowrap;
  }

  .btn-copy:hover {
    background: #4f46e5;
  }

  .btn-add {
    padding: 6px 12px;
    background: #10b981;
    color: white;
    font-size: 14px;
  }

  .btn-add:hover {
    background: #059669;
  }

  .btn-accept {
    padding: 6px 12px;
    background: #10b981;
    color: white;
    font-size: 14px;
  }

  .btn-accept:hover {
    background: #059669;
  }

  .btn-reject {
    padding: 6px 12px;
    background: #dc2626;
    color: white;
    font-size: 14px;
  }

  .btn-reject:hover {
    background: #b91c1c;
  }

  .btn-submit {
    padding: 8px 16px;
    background: #6366f1;
    color: white;
    font-size: 14px;
  }

  .btn-submit:hover:not(:disabled) {
    background: #4f46e5;
  }

  .btn-cancel {
    padding: 8px 16px;
    background: transparent;
    color: #888;
    border: 1px solid #444;
    font-size: 14px;
  }

  .btn-cancel:hover {
    background: #444;
    color: white;
  }

  .btn-send {
    padding: 10px 20px;
    background: #6366f1;
    color: white;
    font-size: 14px;
  }

  .btn-send:hover:not(:disabled) {
    background: #4f46e5;
  }

  .btn-send:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .chat-section {
    width: 100%;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
  }

  .header h2 {
    margin: 0;
  }

  .info-card {
    margin: 20px 0;
    padding: 20px;
    background: #1a1a1a;
    border-radius: 8px;
  }

  .peer-info {
    margin: 10px 0;
    padding: 10px;
    background: #2a2a2a;
    border-radius: 6px;
    font-size: 14px;
  }

  .info-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 10px;
  }

  .info-content {
    flex: 1;
    min-width: 0;
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

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    display: inline-block;
  }

  .status-dot.online {
    background: #10b981;
  }

  .status-dot.offline {
    background: #6b7280;
  }

  .friend-requests {
    margin: 20px 0;
    padding: 15px;
    background: #1a1a1a;
    border-radius: 8px;
  }

  .request-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px;
    margin: 5px 0;
    background: #2a2a2a;
    border-radius: 6px;
  }

  .request-info {
    display: flex;
    flex-direction: column;
  }

  .request-actions {
    display: flex;
    gap: 8px;
  }

  .username {
    color: #888;
    font-size: 14px;
  }

  .main-content {
    display: grid;
    grid-template-columns: 300px 1fr;
    gap: 20px;
    margin-top: 20px;
    height: 600px;
  }

  .friends-sidebar {
    background: #1a1a1a;
    border-radius: 8px;
    padding: 15px;
    display: flex;
    flex-direction: column;
  }

  .sidebar-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 15px;
  }

  .sidebar-header h3 {
    margin: 0;
  }

  .add-friend-form {
    background: #2a2a2a;
    padding: 15px;
    border-radius: 8px;
    margin-bottom: 15px;
  }

  .add-friend-form input {
    width: 100%;
    margin-bottom: 10px;
  }

  .form-actions {
    display: flex;
    gap: 8px;
  }

  .form-actions button {
    flex: 1;
  }

  .friends-list {
    flex: 1;
    overflow-y: auto;
  }

  .friend-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px;
    margin: 5px 0;
    background: #2a2a2a;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.2s;
  }

  .friend-item:hover {
    background: #333;
  }

  .friend-item.selected {
    background: #6366f1;
  }

  .friend-info {
    display: flex;
    flex-direction: column;
  }

  .chat-area {
    background: #1a1a1a;
    border-radius: 8px;
    display: flex;
    flex-direction: column;
  }

  .chat-header {
    padding: 15px;
    border-bottom: 1px solid #444;
  }

  .chat-header h3 {
    margin: 0 0 5px 0;
  }

  .chat-status {
    display: flex;
    align-items: center;
    gap: 5px;
    font-size: 14px;
    color: #888;
  }

  .messages-container {
    flex: 1;
    padding: 20px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .message {
    max-width: 70%;
    padding: 10px 14px;
    border-radius: 12px;
    word-wrap: break-word;
  }

  .message.sent {
    align-self: flex-end;
    background: #6366f1;
    color: white;
  }

  .message.received {
    align-self: flex-start;
    background: #2a2a2a;
    color: white;
  }

  .message-content {
    margin-bottom: 4px;
  }

  .message-meta {
    font-size: 11px;
    opacity: 0.7;
    display: flex;
    align-items: center;
    gap: 5px;
  }

  .status-icon {
    font-size: 10px;
  }

  .message-input-container {
    padding: 15px;
    border-top: 1px solid #444;
    display: flex;
    gap: 10px;
  }

  .message-input {
    flex: 1;
    padding: 10px 14px;
    border: 2px solid #444;
    border-radius: 8px;
    background: #2a2a2a;
    color: #fff;
    font-size: 14px;
  }

  .message-input:focus {
    outline: none;
    border-color: #6366f1;
  }

  .no-selection {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #888;
  }

  .empty-state {
    text-align: center;
    color: #888;
    padding: 20px;
  }
</style>
