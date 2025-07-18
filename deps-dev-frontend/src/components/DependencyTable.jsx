import { useNavigate } from 'react-router-dom';

function DependencyTable({ dependencies, onEdit, onDelete }) {
  const navigate = useNavigate();

  if (!Array.isArray(dependencies) || dependencies.length === 0) {
    return <p className="text-gray-500">No dependencies found</p>;
  }

  return (
    <table className="w-full border mt-4 table-auto text-center">
      <thead>
        <tr>
          <th className="border px-4 py-2">System</th>
          <th className="border px-4 py-2">Name</th>
          <th className="border px-4 py-2">Version</th>
          <th className="border px-4 py-2">Relation</th>
          <th className="border px-4 py-2">Source repository</th>
          <th className="border px-4 py-2">OpenSSF Score</th>
          <th className="border px-4 py-2 w-36">Actions</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-gray-200">
        {dependencies.map((dep) => (
          <tr
            key={`${dep.system}-${dep.name}-${dep.version}`}
            className="cursor-pointer hover:bg-gray-100 divide-x divide-gray-200"
            onClick={() => navigate(`/dependencies/${dep.system}/${dep.name}/${dep.version}`)}
          >
            <td>{dep.system}</td>
            <td>{dep.name}</td>
            <td>{dep.version}</td>
            <td>{dep.relation}</td>
            <td>{dep.source_repo}</td>
            <td>{dep.openssf_score}</td>
            <td onClick={(e) => e.stopPropagation()}>
              <button className="text-blue-600 underline mr-2" onClick={() => onEdit(dep)}>Edit</button>
              <button className="text-red-600 underline" onClick={() => onDelete(dep)}>Delete</button>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export default DependencyTable;
