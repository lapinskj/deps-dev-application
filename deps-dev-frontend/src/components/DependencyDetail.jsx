import { useParams, useNavigate } from 'react-router-dom';
import { useEffect, useState } from 'react';
import DependencyModal from './DependencyModal';

function DependencyDetail() {
  const { system, name, version } = useParams();
  const navigate = useNavigate();
  const [dep, setDep] = useState(null);
  const [modalOpen, setModalOpen] = useState(false);

  useEffect(() => {
    fetch(`${import.meta.env.VITE_API_URL}/dependencies/${system}/${name}/${version}`)
      .then(res => res.json())
      .then(setDep)
      .catch(err => {
        console.error("Failed to fetch dependency", err);
        alert("Could not load dependency.");
      });
  }, [system, name, version]);

  const handleDelete = async () => {
    if (!window.confirm("Delete this dependency?")) return;

    const res = await fetch(`${import.meta.env.VITE_API_URL}/dependencies/${system}/${name}/${version}`, {
      method: "DELETE",
    });

    if (res.ok) {
      alert("Deleted successfully");
      navigate("/");
    } else {
      alert("Delete failed");
    }
  };

    const handleModalSubmit = async (form) => {
    const res = await fetch(`${import.meta.env.VITE_API_URL}/dependencies/${system}/${name}/${version}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
        relation: form.relation || null,
        source_repo: form.source_repo || null,
        openssf_score: form.openssf_score
            ? parseFloat(form.openssf_score)
            : null,
        }),
    });

    if (res.ok) {
        setModalOpen(false);

        try {
        const refreshed = await fetch(`${import.meta.env.VITE_API_URL}/dependencies/${system}/${name}/${version}`);
        const json = await refreshed.json();
        setDep(json);
        } catch (err) {
        console.error("Failed to re-fetch dependency", err);
        alert("Updated, but failed to reload data.");
        }
    } else {
        alert("Update failed");
    }
    };


  if (!dep) return <p className="text-gray-500">Loading...</p>;

  return (
    <div className="max-w-xl mx-auto p-6">
      <DependencyModal
        isOpen={modalOpen}
        onClose={() => setModalOpen(false)}
        onSubmit={handleModalSubmit}
        initialData={dep}
      />

      <h2 className="text-xl font-bold mb-4">Dependency Details</h2>
      <div className="space-y-2 mb-6">
        <p><strong>System:</strong> {dep.system}</p>
        <p><strong>Name:</strong> {dep.name}</p>
        <p><strong>Version:</strong> {dep.version}</p>
        <p><strong>Relation:</strong> {dep.relation || "—"}</p>
        <p><strong>Source Repo:</strong> {dep.source_repo || "—"}</p>
        <p><strong>OpenSSF Score:</strong> {dep.openssf_score ?? "—"}</p>
      </div>

      <div className="flex gap-4">
        <button
          onClick={() => setModalOpen(true)}
          className="bg-blue-600 text-white px-4 py-2 rounded"
        >Edit</button>
        <button
          onClick={handleDelete}
          className="bg-red-600 text-white px-4 py-2 rounded"
        >Delete</button>
        <button
          onClick={() => navigate("/")}
          className="px-4 py-2 border rounded"
        >Back</button>
      </div>
    </div>
  );
}

export default DependencyDetail;
