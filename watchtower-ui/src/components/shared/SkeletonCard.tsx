export default function SkeletonCard() {
  return (
    <div className="bg-card-bg rounded-xl border border-border p-5 shadow-sm shadow-black/10">
      <div className="flex items-start justify-between mb-3">
        <div className="flex-1 space-y-2">
          <div className="h-4 w-3/4 rounded skeleton" />
          <div className="h-3 w-1/2 rounded skeleton" />
        </div>
        <div className="h-5 w-14 rounded-full skeleton" />
      </div>
      <div className="flex items-center justify-between mt-4">
        <div className="h-3 w-20 rounded skeleton" />
        <div className="h-5 w-9 rounded-full skeleton" />
      </div>
    </div>
  );
}
