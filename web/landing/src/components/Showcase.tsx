const gallery = [
  { name: 'Transcript view', img: 'scriberr-transcript page.png' },
  { name: 'Summaries', img: 'scriberr-summarize transcripts.png' },
  { name: 'Notes', img: 'scriberr-annotate transcript and take notes.png' }
];

import Window from './Window';

export default function Showcase() {
  return (
    <section id="showcase" className="container-narrow section">
      <div className="text-center mb-12">
        <span className="eyebrow">Showcase</span>
        <h2 className="text-2xl md:text-3xl font-semibold mt-2">See it in action</h2>
        <p className="subcopy mt-2">A quiet UI that lets your audio take center stage.</p>
      </div>

      <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
        {gallery.map((g) => (
          <figure key={g.name}>
            <Window src={`/screenshots/${g.img}`} alt={g.name} />
            <figcaption className="text-sm text-gray-600 mt-2 text-center">{g.name}</figcaption>
          </figure>
        ))}
      </div>
    </section>
  );
}
