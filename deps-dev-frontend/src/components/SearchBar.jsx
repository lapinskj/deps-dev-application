function SearchBar({ search, setSearch, minScore, setMinScore }) {
    return (
        <div className="flex gap-4 mb-4">
            <input
                type="text"
                placeholder="Search dependencies..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="border px-3 py-2 rounded w-1/2"
            />

            <input
                type="number"
                min="0"
                max="10"
                step="0.1"
                placeholder="Min Score"
                value={minScore}
                onChange={(e) => setMinScore(e.target.value)}
                className="border px-3 py-2 rounded w-32"
            />
        </div>
    );
}

export default SearchBar;
