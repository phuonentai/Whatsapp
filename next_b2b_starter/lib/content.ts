import fs from "node:fs";
import path from "node:path";

/**
 * Build-time content loader for the marketing site (blog + academy).
 *
 * Content lives as markdown files with YAML-ish frontmatter under
 * `content/blog/` and `content/academy/<course>/`. Files are read from disk at
 * build/render time (server components only) — no database, no CMS.
 */

export interface BlogPostMeta {
  slug: string;
  title: string;
  description: string;
  date: string; // ISO date, e.g. "2026-08-01"
  author: string;
  category: string;
  tags: string[];
  cover?: string;
  readingTimeMinutes: number;
  draft?: boolean;
}

export interface BlogPost {
  meta: BlogPostMeta;
  body: string;
}

export interface LessonMeta {
  slug: string;
  order: number;
  title: string;
  description: string;
  durationMinutes: number;
}

export interface Lesson extends LessonMeta {
  body: string;
  prev: LessonMeta | null;
  next: LessonMeta | null;
}

export interface CourseMeta {
  slug: string;
  title: string;
  description: string;
  level: string;
  track: string;
  order: number;
  durationMinutes: number;
  cover?: string;
}

export interface Course {
  meta: CourseMeta;
  lessons: LessonMeta[];
}

const CONTENT_DIR = path.join(process.cwd(), "content");

/** Split `---`-fenced frontmatter from the markdown body. */
export function parseFrontmatter<T>(raw: string): { data: T; body: string } {
  const match = /^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/.exec(raw);
  if (!match) {
    return { data: {} as T, body: raw };
  }
  const [, front, body] = match;
  const data = {} as Record<string, unknown>;
  for (const line of front.split(/\r?\n/)) {
    const eq = line.indexOf(":");
    if (eq === -1) continue;
    const key = line.slice(0, eq).trim();
    let value: unknown = line.slice(eq + 1).trim() as unknown;
    const raw = String(value);
    if ((raw.startsWith('"') && raw.endsWith('"')) || (raw.startsWith("'") && raw.endsWith("'"))) {
      value = raw.slice(1, -1);
    } else if (raw === "true") {
      value = true;
    } else if (raw === "false") {
      value = false;
    } else if (/^-?\d+$/.test(raw)) {
      value = Number(raw);
    } else if (raw.startsWith("[") && raw.endsWith("]")) {
      value = raw
        .slice(1, -1)
        .split(",")
        .map((v) => v.trim().replace(/^["']|["']$/g, ""))
        .filter(Boolean);
    }
    data[key] = value;
  }
  return { data: data as T, body };
}

function estimateReadingTime(body: string): number {
  const words = body.trim().split(/\s+/).filter(Boolean).length;
  return Math.max(1, Math.round(words / 200));
}

function listMarkdownFiles(dir: string): string[] {
  if (!fs.existsSync(dir)) return [];
  return fs
    .readdirSync(dir, { withFileTypes: true })
    .filter((e) => e.isFile() && e.name.endsWith(".md"))
    .map((e) => path.join(dir, e.name));
}

// --- Blog ---------------------------------------------------------------

export function getBlogPosts(): BlogPostMeta[] {
  const files = listMarkdownFiles(path.join(CONTENT_DIR, "blog"));
  const posts: BlogPostMeta[] = [];
  for (const file of files) {
    const raw = fs.readFileSync(file, "utf8");
    const { data, body } = parseFrontmatter<Partial<BlogPostMeta>>(raw);
    if (data.draft) continue;
    const slug = path.basename(file, ".md");
    posts.push({
      slug,
      title: data.title ?? slug,
      description: data.description ?? "",
      date: data.date ?? "",
      author: data.author ?? "Equipo NexoChat",
      category: data.category ?? "General",
      tags: data.tags ?? [],
      cover: data.cover,
      readingTimeMinutes: estimateReadingTime(body),
    });
  }
  return posts.sort((a, b) => b.date.localeCompare(a.date));
}

export function getBlogPost(slug: string): BlogPost | null {
  const file = path.join(CONTENT_DIR, "blog", `${slug}.md`);
  if (!fs.existsSync(file)) return null;
  const raw = fs.readFileSync(file, "utf8");
  const { data, body } = parseFrontmatter<Partial<BlogPostMeta>>(raw);
  const meta: BlogPostMeta = {
    slug,
    title: data.title ?? slug,
    description: data.description ?? "",
    date: data.date ?? "",
    author: data.author ?? "Equipo NexoChat",
    category: data.category ?? "General",
    tags: data.tags ?? [],
    cover: data.cover,
    readingTimeMinutes: estimateReadingTime(body),
  };
  return { meta, body };
}

export function getBlogCategories(): string[] {
  return [...new Set(getBlogPosts().map((p) => p.category))].sort();
}

// --- Academy ------------------------------------------------------------

function courseDir(slug: string): string {
  return path.join(CONTENT_DIR, "academy", slug);
}

export function getCourses(): CourseMeta[] {
  const root = path.join(CONTENT_DIR, "academy");
  if (!fs.existsSync(root)) return [];
  const courses: CourseMeta[] = [];
  for (const entry of fs.readdirSync(root, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const courseFile = path.join(root, entry.name, "course.md");
    if (!fs.existsSync(courseFile)) continue;
    const raw = fs.readFileSync(courseFile, "utf8");
    const { data } = parseFrontmatter<Partial<CourseMeta>>(raw);
    const lessons = listMarkdownFiles(courseDir(entry.name))
      .map((f) => path.basename(f, ".md"))
      .filter((name) => name !== "course");
    const durationMinutes =
      data.durationMinutes ??
      lessons.reduce((sum, slug) => {
        const lesson = getLessonMeta(entry.name, slug);
        return sum + (lesson?.durationMinutes ?? 0);
      }, 0);
    courses.push({
      slug: entry.name,
      title: data.title ?? entry.name,
      description: data.description ?? "",
      level: data.level ?? "Principiante",
      track: data.track ?? "General",
      order: data.order ?? 0,
      durationMinutes,
      cover: data.cover,
    });
  }
  return courses.sort((a, b) => a.order - b.order || a.title.localeCompare(b.title));
}

function getLessonMeta(courseSlug: string, lessonFile: string): LessonMeta | null {
  const file = path.join(courseDir(courseSlug), `${lessonFile}.md`);
  if (!fs.existsSync(file)) return null;
  const raw = fs.readFileSync(file, "utf8");
  const { data } = parseFrontmatter<Partial<LessonMeta>>(raw);
  return {
    slug: lessonFile,
    order: data.order ?? 0,
    title: data.title ?? lessonFile,
    description: data.description ?? "",
    durationMinutes: data.durationMinutes ?? 0,
  };
}

export function getCourse(slug: string): Course | null {
  const courseFile = path.join(courseDir(slug), "course.md");
  if (!fs.existsSync(courseFile)) return null;
  const raw = fs.readFileSync(courseFile, "utf8");
  const { data } = parseFrontmatter<Partial<CourseMeta>>(raw);
  const lessons = listMarkdownFiles(courseDir(slug))
    .map((f) => path.basename(f, ".md"))
    .filter((name) => name !== "course")
    .map((name) => getLessonMeta(slug, name))
    .filter((l): l is LessonMeta => l !== null)
    .sort((a, b) => a.order - b.order || a.slug.localeCompare(b.slug));
  const meta: CourseMeta = {
    slug,
    title: data.title ?? slug,
    description: data.description ?? "",
    level: data.level ?? "Principiante",
    track: data.track ?? "General",
    order: data.order ?? 0,
    durationMinutes: data.durationMinutes ?? lessons.reduce((s, l) => s + l.durationMinutes, 0),
    cover: data.cover,
  };
  return { meta, lessons };
}

export function getLesson(courseSlug: string, lessonSlug: string): Lesson | null {
  const course = getCourse(courseSlug);
  if (!course) return null;
  const file = path.join(courseDir(courseSlug), `${lessonSlug}.md`);
  if (!fs.existsSync(file)) return null;
  const raw = fs.readFileSync(file, "utf8");
  const { data, body } = parseFrontmatter<Partial<LessonMeta>>(raw);
  const index = course.lessons.findIndex((l) => l.slug === lessonSlug);
  const meta: LessonMeta = {
    slug: lessonSlug,
    order: data.order ?? (index >= 0 ? course.lessons[index].order : 0),
    title: data.title ?? lessonSlug,
    description: data.description ?? "",
    durationMinutes: data.durationMinutes ?? 0,
  };
  return {
    ...meta,
    body,
    prev: index > 0 ? course.lessons[index - 1] : null,
    next: index >= 0 && index < course.lessons.length - 1 ? course.lessons[index + 1] : null,
  };
}
