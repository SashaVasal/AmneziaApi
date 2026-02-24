import { useEffect, useMemo, useState } from "react";
import QRCode from "react-qr-code";

const API_BASE = import.meta.env.VITE_API_BASE_URL || "/api";

function formatBytes(value) {
  const n = Number(value || 0);
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / (1024 * 1024)).toFixed(1)} MB`;
  return `${(n / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

function authHeaders(token) {
  return {
    Authorization: `Bearer ${token}`,
    "Content-Type": "application/json"
  };
}

export default function App() {
  const [token, setToken] = useState(() => localStorage.getItem("awg_token") || "");
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [peers, setPeers] = useState([]);
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [configPeer, setConfigPeer] = useState(null);
  const [configText, setConfigText] = useState("");

  const loggedIn = useMemo(() => !!token, [token]);

  async function api(path, options = {}) {
    const res = await fetch(`${API_BASE}${path}`, options);
    if (!res.ok) {
      const text = await res.text();
      throw new Error(text || `HTTP ${res.status}`);
    }
    const ct = res.headers.get("content-type") || "";
    if (ct.includes("application/json")) return res.json();
    return res.text();
  }

  async function doLogin(e) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const data = await api("/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password })
      });
      localStorage.setItem("awg_token", data.token);
      setToken(data.token);
      setPassword("");
    } catch (err) {
      setError("Ошибка авторизации");
    } finally {
      setLoading(false);
    }
  }

  async function refreshData() {
    if (!token) return;
    setLoading(true);
    setError("");
    try {
      const [peerRes, statsRes] = await Promise.all([
        api("/peers", { headers: authHeaders(token) }),
        api("/stats", { headers: authHeaders(token) })
      ]);
      setPeers(peerRes.items || []);
      setStats(statsRes);
    } catch (err) {
      setError("Не удалось загрузить данные");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    refreshData();
  }, [token]);

  async function createPeer(e) {
    e.preventDefault();
    if (!name.trim()) return;
    setLoading(true);
    setError("");
    try {
      await api("/peers", {
        method: "POST",
        headers: authHeaders(token),
        body: JSON.stringify({ name: name.trim() })
      });
      setName("");
      await refreshData();
    } catch (err) {
      setError("Не удалось создать peer");
    } finally {
      setLoading(false);
    }
  }

  async function removePeer(id) {
    if (!confirm("Удалить peer?")) return;
    setLoading(true);
    setError("");
    try {
      await api(`/peers/${id}`, {
        method: "DELETE",
        headers: authHeaders(token)
      });
      if (configPeer?.id === id) {
        setConfigPeer(null);
        setConfigText("");
      }
      await refreshData();
    } catch (err) {
      setError("Не удалось удалить peer");
    } finally {
      setLoading(false);
    }
  }

  async function openConfig(peer) {
    setLoading(true);
    setError("");
    try {
      const cfg = await api(`/peers/${peer.id}/config`, {
        headers: { Authorization: `Bearer ${token}` }
      });
      setConfigPeer(peer);
      setConfigText(cfg);
    } catch (err) {
      setError("Не удалось получить конфиг");
    } finally {
      setLoading(false);
    }
  }

  function downloadConfig() {
    if (!configPeer || !configText) return;
    const blob = new Blob([configText], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${configPeer.name || configPeer.id}.conf`;
    a.click();
    URL.revokeObjectURL(url);
  }

  function logout() {
    localStorage.removeItem("awg_token");
    setToken("");
    setPeers([]);
    setStats(null);
    setConfigPeer(null);
    setConfigText("");
  }

  if (!loggedIn) {
    return (
      <main className="mx-auto mt-24 w-full max-w-md rounded-xl border border-slate-800 bg-slate-900 p-6 shadow-xl">
        <h1 className="mb-6 text-2xl font-semibold">AmneziaWG Manager</h1>
        <form onSubmit={doLogin} className="space-y-4">
          <input
            className="w-full rounded border border-slate-700 bg-slate-950 px-3 py-2"
            placeholder="Username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
          />
          <input
            className="w-full rounded border border-slate-700 bg-slate-950 px-3 py-2"
            placeholder="Password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <button
            className="w-full rounded bg-emerald-600 px-3 py-2 font-medium hover:bg-emerald-500 disabled:opacity-50"
            disabled={loading}
          >
            {loading ? "Вход..." : "Войти"}
          </button>
        </form>
        {error && <p className="mt-4 text-sm text-red-400">{error}</p>}
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-7xl p-6">
      <header className="mb-6 flex items-center justify-between">
        <h1 className="text-3xl font-bold">AmneziaWG Dashboard</h1>
        <button
          className="rounded border border-slate-700 px-3 py-2 hover:bg-slate-800"
          onClick={logout}
        >
          Выйти
        </button>
      </header>

      {error && <div className="mb-4 rounded border border-red-900 bg-red-950/40 p-3 text-red-300">{error}</div>}

      <section className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-3">
        <div className="rounded-lg border border-slate-800 bg-slate-900 p-4">
          <p className="text-sm text-slate-400">Всего peers</p>
          <p className="mt-1 text-2xl font-semibold">{stats?.total_peers ?? "-"}</p>
        </div>
        <div className="rounded-lg border border-slate-800 bg-slate-900 p-4">
          <p className="text-sm text-slate-400">Активные peers</p>
          <p className="mt-1 text-2xl font-semibold">{stats?.active_peers ?? "-"}</p>
        </div>
        <div className="rounded-lg border border-slate-800 bg-slate-900 p-4">
          <p className="text-sm text-slate-400">Трафик</p>
          <p className="mt-1 text-sm">
            RX: {formatBytes(stats?.rx_bytes)} / TX: {formatBytes(stats?.tx_bytes)}
          </p>
        </div>
      </section>

      <section className="mb-6 rounded-lg border border-slate-800 bg-slate-900 p-4">
        <form onSubmit={createPeer} className="flex flex-col gap-3 md:flex-row">
          <input
            className="flex-1 rounded border border-slate-700 bg-slate-950 px-3 py-2"
            placeholder="Имя клиента"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <button
            className="rounded bg-emerald-600 px-4 py-2 font-medium hover:bg-emerald-500 disabled:opacity-50"
            disabled={loading}
          >
            Создать
          </button>
          <button
            type="button"
            className="rounded border border-slate-700 px-4 py-2 hover:bg-slate-800"
            onClick={refreshData}
          >
            Обновить
          </button>
        </form>
      </section>

      <section className="overflow-hidden rounded-lg border border-slate-800 bg-slate-900">
        <table className="min-w-full text-left text-sm">
          <thead className="bg-slate-800/60 text-slate-300">
            <tr>
              <th className="px-4 py-3">Имя</th>
              <th className="px-4 py-3">Allowed IP</th>
              <th className="px-4 py-3">RX</th>
              <th className="px-4 py-3">TX</th>
              <th className="px-4 py-3">Handshake</th>
              <th className="px-4 py-3">Действия</th>
            </tr>
          </thead>
          <tbody>
            {peers.map((peer) => (
              <tr key={peer.id} className="border-t border-slate-800">
                <td className="px-4 py-3">{peer.name}</td>
                <td className="px-4 py-3">{peer.allowed_ip}</td>
                <td className="px-4 py-3">{formatBytes(peer.rx_bytes)}</td>
                <td className="px-4 py-3">{formatBytes(peer.tx_bytes)}</td>
                <td className="px-4 py-3">
                  {peer.last_handshake ? new Date(peer.last_handshake).toLocaleString() : "-"}
                </td>
                <td className="px-4 py-3">
                  <div className="flex gap-2">
                    <button
                      className="rounded border border-slate-600 px-2 py-1 hover:bg-slate-800"
                      onClick={() => openConfig(peer)}
                    >
                      Конфиг/QR
                    </button>
                    <button
                      className="rounded border border-red-800 px-2 py-1 text-red-300 hover:bg-red-950/50"
                      onClick={() => removePeer(peer.id)}
                    >
                      Удалить
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {peers.length === 0 && (
              <tr>
                <td className="px-4 py-6 text-slate-400" colSpan={6}>
                  Peers не найдены
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </section>

      {configPeer && (
        <section className="mt-6 grid grid-cols-1 gap-6 rounded-lg border border-slate-800 bg-slate-900 p-4 lg:grid-cols-2">
          <div>
            <h2 className="mb-2 text-lg font-semibold">Конфиг: {configPeer.name}</h2>
            <pre className="max-h-96 overflow-auto rounded bg-slate-950 p-3 text-xs">{configText}</pre>
            <button
              className="mt-3 rounded bg-emerald-600 px-3 py-2 font-medium hover:bg-emerald-500"
              onClick={downloadConfig}
            >
              Скачать .conf
            </button>
          </div>
          <div className="flex flex-col items-center">
            <h2 className="mb-2 text-lg font-semibold">QR Code</h2>
            <div className="rounded bg-white p-4">
              <QRCode value={configText || "empty"} size={240} />
            </div>
          </div>
        </section>
      )}
    </main>
  );
}
