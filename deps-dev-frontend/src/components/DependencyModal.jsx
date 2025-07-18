import { useState, useEffect } from "react";

function DependencyModal({ isOpen, onClose, onSubmit, initialData = null }) {
  const [form, setForm] = useState({
    system: "",
    name: "",
    version: "",
    relation: "",
    source_repo: "",
    openssf_score: "",
  });

  useEffect(() => {
    if (initialData) {
      setForm({
        system: initialData.system || "",
        name: initialData.name || "",
        version: initialData.version || "",
        relation: initialData.relation || "",
        source_repo: initialData.source_repo || "",
        openssf_score: initialData.openssf_score ?? "",
      });
    } else {
      setForm({
        system: "",
        name: "",
        version: "",
        relation: "",
        source_repo: "",
        openssf_score: "",
      });
    }
  }, [initialData]);


  const handleChange = (e) => {
    setForm((prev) => ({
      ...prev,
      [e.target.name]: e.target.value,
    }));
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    onSubmit(form);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center">
      <div className="bg-white p-6 rounded shadow-lg w-full max-w-xl">
        <h2 className="text-lg font-semibold mb-4">
        {initialData ? "Edit Dependency" : "Add New Dependency"}
        </h2>


        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="flex gap-2">
            <input
              name="system"
              value={form.system}
              onChange={handleChange}
              placeholder="System"
              className="border p-2 rounded w-1/3"
              required
              disabled={!!initialData}
            />
            <input
              name="name"
              value={form.name}
              onChange={handleChange}
              placeholder="Name"
              className="border p-2 rounded w-1/3"
              required
              disabled={!!initialData}
            />
            <input
              name="version"
              value={form.version}
              onChange={handleChange}
              placeholder="Version"
              className="border p-2 rounded w-1/3"
              required
              disabled={!!initialData}
            />
          </div>

          <input
            name="relation"
            value={form.relation}
            onChange={handleChange}
            placeholder="Relation (optional)"
            className="border p-2 rounded w-full"
          />
          <input
            name="source_repo"
            value={form.source_repo}
            onChange={handleChange}
            placeholder="Source Repo (optional)"
            className="border p-2 rounded w-full"
          />
          <input
            name="openssf_score"
            value={form.openssf_score}
            onChange={handleChange}
            placeholder="OpenSSF Score (0–10)"
            type="number"
            min="0"
            max="10"
            step="0.1"
            className="border p-2 rounded w-full"
          />

          <div className="flex justify-end gap-4">
            <button type="button" onClick={onClose} className="px-4 py-2 border rounded">
              Cancel
            </button>
            <button type="submit" className="bg-blue-600 text-white px-4 py-2 rounded">
              Save
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default DependencyModal;
