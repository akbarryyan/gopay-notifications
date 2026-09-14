/**
 * Minimal syntax presentation for a static snippet.
 *
 * Deliberately not a syntax highlighter: it only tints comment lines and
 * double-quoted strings, which is enough to make a shell snippet readable
 * without pulling a highlighting library into the landing page bundle.
 */
function renderLine(line: string, key: number) {
  if (line.trimStart().startsWith("#")) {
    return (
      <span key={key} className="block text-slate-500">
        {line || " "}
      </span>
    );
  }

  // Split on double-quoted runs, keeping the delimiters as their own parts.
  const parts = line.split(/("(?:[^"\\]|\\.)*")/g);

  return (
    <span key={key} className="block">
      {parts.map((part, i) =>
        part.startsWith('"') && part.endsWith('"') && part.length > 1 ? (
          <span key={i} className="text-amber-300">
            {part}
          </span>
        ) : (
          <span key={i}>{part}</span>
        ),
      )}
      {line === "" && " "}
    </span>
  );
}

export default function CodeBlock({
  title,
  code,
}: {
  title: string;
  code: string;
}) {
  return (
    <div className="overflow-hidden rounded-xl border border-white/10 bg-brand-navy-dark/80 shadow-2xl">
      <div className="flex items-center gap-2 border-b border-white/10 px-4 py-3">
        <span className="h-2.5 w-2.5 rounded-full bg-rose-400/70" />
        <span className="h-2.5 w-2.5 rounded-full bg-brand-yellow/70" />
        <span className="h-2.5 w-2.5 rounded-full bg-teal-400/70" />
        <span className="ml-2 text-xs font-medium text-white/50">{title}</span>
      </div>
      <pre className="overflow-x-auto px-4 py-4 text-[13px] leading-relaxed text-white/90">
        <code className="font-mono">
          {code.split("\n").map((line, i) => renderLine(line, i))}
        </code>
      </pre>
    </div>
  );
}
