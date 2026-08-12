import Link from "next/link";
import { ArrowLeft, Home } from "lucide-react";
import { Button } from "@/components/ui/button";

/** Branded 404 for the whole app (marketing + product), soft business palette. */
export default function NotFound() {
  return (
    <div className="min-h-screen bg-background text-foreground flex items-center justify-center px-4">
      <div className="max-w-xl text-center py-24">
        <p className="font-heading text-7xl sm:text-8xl font-bold text-primary mb-4">404</p>
        <h1 className="font-heading text-3xl sm:text-4xl font-bold tracking-tight mb-4">
          Página no encontrada
        </h1>
        <p className="text-lg text-muted-foreground leading-relaxed mx-auto mb-10">
          La página que buscas no existe o fue movida. Revisa la URL o vuelve al inicio.
        </p>
        <div className="flex flex-col sm:flex-row gap-4 justify-center">
          <Link href="/">
            <Button className="bg-primary hover:bg-primary/90 text-primary-foreground px-6 py-3 rounded-xl font-semibold">
              <Home className="w-4 h-4 mr-2" />
              Volver al inicio
            </Button>
          </Link>
          <Link href="/blog">
            <Button variant="outline" className="border-2 border-border hover:border-primary text-foreground px-6 py-3 rounded-xl font-semibold hover:bg-primary/10">
              <ArrowLeft className="w-4 h-4 mr-2" />
              Ir al blog
            </Button>
          </Link>
        </div>
      </div>
    </div>
  );
}
