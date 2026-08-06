import Script from "next/script";

type JsonLdProps = {
  data: Record<string, unknown> | Record<string, unknown>[];
  id?: string;
};

export function JsonLd({ data, id }: JsonLdProps) {
  const json = Array.isArray(data) ? data : [data];
  const content =
    json.length === 1 ? JSON.stringify(json[0]) : JSON.stringify(json);
  return (
    <Script
      id={id}
      type="application/ld+json"
      strategy="afterInteractive"
      dangerouslySetInnerHTML={{ __html: content }}
    />
  );
}
