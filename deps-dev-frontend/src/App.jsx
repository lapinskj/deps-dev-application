import { useEffect, useState } from 'react';
import SearchBar from './components/SearchBar';
import DependencyTable from './components/DependencyTable';
import ScoreChart from './components/ScoreChart';
import DependencyModal from './components/DependencyModal';

function App() {
  const [deps, setDeps] = useState([]);
  const [search, setSearch] = useState("");
  const [minScore, setMinScore] = useState("");

const [modalOpen, setModalOpen] = useState(false);
const [editTarget, setEditTarget] = useState(null);

const handleCreateClick = () => {
  setEditTarget(null);
  setModalOpen(true);
};

const handleEditClick = (dep) => {
  setEditTarget(dep);
  setModalOpen(true);
};

  const handleModalSubmit = async (form) => {
    const method = editTarget ? "PUT" : "POST";
    const url = editTarget
      ? `${import.meta.env.VITE_API_URL}/dependencies/${form.system}/${form.name}/${form.version}`
      : `${import.meta.env.VITE_API_URL}/dependencies`;

    const body = {
      system: form.system,
      name: form.name,
      version: form.version,
      relation: form.relation || null,
      source_repo: form.source_repo || null,
      openssf_score: form.openssf_score
        ? parseFloat(form.openssf_score)
        : null,
    };

    const res = await fetch(url, {
      method,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });

    if (res.ok) {
      setModalOpen(false);
      const updated = await fetchDeps(search, minScore);
      setDeps(updated);
    } else {
      alert("Failed to save dependency");
    }
  };

  const handleDelete = async (dep) => {
  if (!window.confirm(`Delete ${dep.name}@${dep.version}?`)) return;

  try {
    const res = await fetch(
      `${import.meta.env.VITE_API_URL}/dependencies/${dep.system}/${dep.name}/${dep.version}`,
      { method: "DELETE" }
    );

    if (!res.ok) throw new Error("Delete failed");

    const updated = await fetchDeps(search, minScore);
    setDeps(updated);
  } catch (err) {
    console.error(err);
    alert("Failed to delete dependency");
  }
};


useEffect(() => {
  const controller = new AbortController();
  const timeout = setTimeout(() => {
    fetchDeps(search, minScore, controller.signal)
      .then(setDeps)
      .catch((err) => {
        if (err.name !== "AbortError") console.error("Fetch failed:", err);
      });
  }, 500);

  return () => {
    clearTimeout(timeout);
    controller.abort();
  };
}, [search, minScore]);

const filtered = deps;

const handleRefresh = async () => {
  try {
    const res = await fetch(`${import.meta.env.VITE_API_URL}/dependencies/refresh`, {
      method: "POST",
    });

    if (!res.ok) {
      throw new Error("Failed to refresh data");
    }

    alert("Data refresh triggered!");

    const data = await fetchDeps(search, minScore);
    setDeps(data);
  } catch (err) {
    console.error(err);
    alert("Failed to refresh data");
  }
};

const fetchDeps = async (search, minScore, signal) => {
  let url = `${import.meta.env.VITE_API_URL}/dependencies`;
  const params = new URLSearchParams();
  if (search) params.append("name", search);
  if (minScore) params.append("min_score", minScore);
  const queryString = params.toString();
  if (queryString) url += `?${queryString}`;

  const res = await fetch(url, { signal });
  return res.json();
};

  return (
    <div className="max-w-4xl mx-auto p-6">
      <DependencyModal
        isOpen={modalOpen}
        onClose={() => setModalOpen(false)}
        onSubmit={handleModalSubmit}
        initialData={editTarget}
      />

     <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">Dependency Dashboard</h1>
        <div className="flex gap-3">
          <button
            onClick={handleCreateClick}
            className="bg-green-600 text-white px-4 py-2 rounded hover:bg-green-700"
          >
            Add Dependency
          </button>

          <button
            onClick={handleRefresh}
            className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700"
          >
            Refresh Data
          </button>
        </div>
      </div>

      <SearchBar search={search} setSearch={setSearch} minScore={minScore} setMinScore={setMinScore}/>
      <DependencyTable dependencies={filtered} onEdit={handleEditClick} onDelete={handleDelete}/>
      <ScoreChart data={(filtered || []).filter(d => d.openssf_score != null)} />
    </div>
  );
}

export default App;
