import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';

function ScoreChart({ data }) {
  return (
    <div className="h-64 mt-8">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={data}>
          <XAxis dataKey="name" />
          <YAxis domain={[0, 10]} />
          <Tooltip />
          <Bar dataKey="openssf_score" barSize={50}  fill="#3b82f6"/>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

export default ScoreChart;
