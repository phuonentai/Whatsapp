"use client";

export function UpgradeBanner({ feature, plan }: { feature: string; plan: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <div className="text-4xl mb-4 text-gray-300">🔒</div>
      <h3 className="text-lg font-semibold text-gray-700 mb-2">{feature} es una funcionalidad {plan}</h3>
      <p className="text-gray-500 mb-6">Actualiza tu plan para acceder a esta funcionalidad.</p>
      <a
        href="/dashboard/settings?view=suscripcion"
        className="bg-emerald-500 text-white px-6 py-2 rounded-lg hover:bg-emerald-600 transition"
      >
        Actualizar a {plan}
      </a>
    </div>
  );
}
