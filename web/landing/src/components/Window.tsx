type Props = {
  src: string;
  alt?: string;
  className?: string;
};

export default function Window({ src, alt, className }: Props) {
  return (
    <div className={`rounded-2xl shadow-soft overflow-hidden bg-white hover-lift ${className ?? ''}`}>
      <div className="flex items-center gap-2 px-3 py-2 bg-gray-100">
        <span className="size-3 rounded-full bg-red-400/80" />
        <span className="size-3 rounded-full bg-yellow-400/80" />
        <span className="size-3 rounded-full bg-green-400/80" />
      </div>
      <img src={src} alt={alt} className="w-full object-cover" />
    </div>
  );
}
