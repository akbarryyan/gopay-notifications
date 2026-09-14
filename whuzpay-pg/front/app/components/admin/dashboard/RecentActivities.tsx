const activities = [
  {
    time: "42 Mins Ago",
    title: "Task Updated",
    detail: "Nikolai Updated a Task",
    color: "bg-[#6366f1]",
  },
  {
    time: "1 day Ago",
    title: "Deal Added",
    detail: "Panshi Updated a Task",
    color: "bg-[#ec4899]",
  },
  {
    time: "42 Mins Ago",
    title: "Published Article",
    detail: "Rasel Published a Article",
    color: "bg-[#06b6d4]",
  },
  {
    time: "1 day Ago",
    title: "Dock Updated",
    detail: "Reshml Updated a Dock",
    color: "bg-[#eab308]",
  },
];

export default function RecentActivities() {
  return (
    <div className="h-full rounded-xl bg-white p-5 shadow-[0_1px_3px_rgba(16,38,73,0.06)]">
      <h3 className="text-[15px] font-semibold text-[#1f2a37]">Recent Activities</h3>

      <ol className="relative mt-5 space-y-0">
        {activities.map((item, i) => (
          <li key={`${item.title}-${i}`} className="relative flex gap-4 pb-6 last:pb-0">
            {/* timeline line */}
            {i < activities.length - 1 && (
              <span className="absolute left-[59px] top-6 bottom-0 w-px bg-[#e8eef4] sm:left-[67px]" />
            )}

            <span className="w-[52px] shrink-0 pt-0.5 text-right text-[11px] leading-tight text-[#8a97a8] sm:w-[60px]">
              {item.time}
            </span>

            <span
              className={`relative z-10 mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full ${item.color}`}
            >
              <span className="h-1.5 w-1.5 rounded-full bg-white" />
            </span>

            <div className="min-w-0 pt-0">
              <p className="text-[13px] font-semibold text-[#1f2a37]">{item.title}</p>
              <p className="mt-0.5 text-[12px] text-[#8a97a8]">{item.detail}</p>
            </div>
          </li>
        ))}
      </ol>
    </div>
  );
}
