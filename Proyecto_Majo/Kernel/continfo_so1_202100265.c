// continfo_so1_202100265.c
// Módulo para /proc que muestra info de procesos de contenedores
// Compatible con kernels recientes (6.x) y gcc-13
#include <linux/version.h>
#include <linux/init.h>
#include <linux/module.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/sched.h>
#include <linux/sched/signal.h>
#include <linux/sched/mm.h>
#include <linux/compiler.h>     // READ_ONCE
#include <linux/timekeeping.h>  // ktime_get_real_ts64
#include <linux/time64.h>       // time64_to_tm
#include <linux/mm.h>
#include <linux/sysinfo.h>
#include <linux/string.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("202100265");
MODULE_DESCRIPTION("SO1 - continfo a /proc/continfo_so1_202100265");
MODULE_VERSION("1.0");

#define PROC_FILENAME "continfo_so1_202100265"

/* ---------------------------------------------------------
 * Helpers para información de memoria
 * ---------------------------------------------------------*/

static void get_memory_info(unsigned long *total_kb, unsigned long *free_kb, unsigned long *used_kb)
{
    struct sysinfo si;
    
    si_meminfo(&si);
    
    *total_kb = si.totalram * (PAGE_SIZE / 1024);
    *free_kb = si.freeram * (PAGE_SIZE / 1024);
    *used_kb = *total_kb - *free_kb;
}

/* ---------------------------------------------------------
 * Helpers para procesos
 * ---------------------------------------------------------*/

// Obtiene VSZ en KB (memoria virtual)
static unsigned long get_vsz_kb(struct task_struct *task)
{
    struct mm_struct *mm;
    unsigned long vsz = 0;
    
    if (!task)
        return 0;
        
    mm = get_task_mm(task);
    if (mm) {
        vsz = mm->total_vm * (PAGE_SIZE / 1024);
        mmput(mm);
    }
    
    return vsz;
}

// Obtiene RSS en KB (memoria física)
static unsigned long get_rss_kb(struct task_struct *task)
{
    struct mm_struct *mm;
    unsigned long rss = 0;
    
    if (!task)
        return 0;
        
    mm = get_task_mm(task);
    if (mm) {
        rss = get_mm_rss(mm) * (PAGE_SIZE / 1024);
        mmput(mm);
    }
    
    return rss;
}

// Calcula porcentaje de memoria (aproximado)
static int get_memory_percent(unsigned long rss_kb)
{
    struct sysinfo si;
    unsigned long total_kb;
    
    si_meminfo(&si);
    total_kb = si.totalram * (PAGE_SIZE / 1024);
    
    if (total_kb == 0)
        return 0;
        
    return (int)((rss_kb * 100) / total_kb);
}

// Heurística mejorada de "porcentaje" CPU según estado de tarea
static int get_cpu_percent(struct task_struct *task)
{
    unsigned long st;

    if (!task)
        return 0;

    if (task_is_running(task))
        return 3;

    st = READ_ONCE(task->__state);
    switch (st) {
    case TASK_INTERRUPTIBLE:
    case TASK_UNINTERRUPTIBLE:
        return 1;
    default:
        return 0;
    }
}

// Determina si un proceso es de contenedor (versión simplificada)
static bool is_container_process(struct task_struct *task)
{
    if (!task)
        return false;

    // Verificar si el comando contiene palabras clave de contenedores
    if (strstr(task->comm, "docker") || 
        strstr(task->comm, "containerd") ||
        strstr(task->comm, "runc") ||
        strstr(task->comm, "pause") ||
        strstr(task->comm, "container") ||
        strstr(task->comm, "podman") ||
        strstr(task->comm, "cri-o") ||
        strstr(task->comm, "shim"))
        return true;

    // Heurística adicional: procesos con PPID de containerd o dockerd
    if (task->real_parent) {
        if (strstr(task->real_parent->comm, "containerd") ||
            strstr(task->real_parent->comm, "dockerd") ||
            strstr(task->real_parent->comm, "docker"))
            return true;
    }
        
    return false;
}

// Obtiene la línea de comando o información del contenedor
static void get_task_cmdline_or_container_id(struct task_struct *task, char *buffer, size_t size)
{
    struct mm_struct *mm;
    
    if (!task || !buffer || size == 0) {
        if (buffer && size > 0)
            buffer[0] = '\0';
        return;
    }
    
    // Intentar obtener cmdline
    mm = get_task_mm(task);
    if (mm) {
        // En un caso real, aquí leeríamos /proc/[pid]/cmdline
        // Por simplicidad, usamos el comm
        snprintf(buffer, size, "%s", task->comm);
        mmput(mm);
    } else {
        snprintf(buffer, size, "[%s]", task->comm);
    }
}

/* ---------------------------------------------------------
 * /proc printer
 * ---------------------------------------------------------*/

static int continfo_show(struct seq_file *m, void *v)
{
    struct timespec64 ts;
    struct tm tm;
    unsigned long total_ram_kb, free_ram_kb, used_ram_kb;

    // Fecha/hora en runtime
    ktime_get_real_ts64(&ts);
    time64_to_tm(ts.tv_sec, 0, &tm);
    
    // Información de memoria del sistema
    get_memory_info(&total_ram_kb, &free_ram_kb, &used_ram_kb);
    
    // Formato JSON para facilitar el parseo en GO
    seq_puts(m, "{\n");
    seq_printf(m, "  \"timestamp\": \"%04ld-%02d-%02d %02d:%02d:%02d\",\n",
               (long)tm.tm_year + 1900, tm.tm_mon + 1, tm.tm_mday,
               tm.tm_hour, tm.tm_min, tm.tm_sec);
    
    seq_printf(m, "  \"memory\": {\n");
    seq_printf(m, "    \"total_kb\": %lu,\n", total_ram_kb);
    seq_printf(m, "    \"free_kb\": %lu,\n", free_ram_kb);
    seq_printf(m, "    \"used_kb\": %lu\n", used_ram_kb);
    seq_printf(m, "  },\n");
    
    seq_puts(m, "  \"containers\": [\n");

    rcu_read_lock();
    {
        struct task_struct *t;
        bool first = true;
        
        for_each_process(t) {
            if (is_container_process(t)) {
                unsigned long vsz_kb = get_vsz_kb(t);
                unsigned long rss_kb = get_rss_kb(t);
                int cpu_pct = get_cpu_percent(t);
                int mem_pct = get_memory_percent(rss_kb);
                char cmdline[256];
                
                get_task_cmdline_or_container_id(t, cmdline, sizeof(cmdline));
                
                if (!first)
                    seq_puts(m, ",\n");
                first = false;
                
                seq_puts(m, "    {\n");
                seq_printf(m, "      \"pid\": %d,\n", t->pid);
                seq_printf(m, "      \"ppid\": %d,\n", 
                          t->real_parent ? t->real_parent->pid : 0);
                seq_printf(m, "      \"name\": \"%s\",\n", t->comm);
                seq_printf(m, "      \"cmdline\": \"%s\",\n", cmdline);
                seq_printf(m, "      \"vsz_kb\": %lu,\n", vsz_kb);
                seq_printf(m, "      \"rss_kb\": %lu,\n", rss_kb);
                seq_printf(m, "      \"memory_percent\": %d,\n", mem_pct);
                seq_printf(m, "      \"cpu_percent\": %d\n", cpu_pct);
                seq_puts(m, "    }");
            }
        }
    }
    rcu_read_unlock();

    seq_puts(m, "\n  ]\n");
    seq_puts(m, "}\n");

    return 0;
}

/* ---------------------------------------------------------
 * /proc plumbing
 * ---------------------------------------------------------*/

static int continfo_open(struct inode *inode, struct file *file)
{
    return single_open(file, continfo_show, NULL);
}

#if LINUX_VERSION_CODE >= KERNEL_VERSION(5,6,0)
static const struct proc_ops continfo_proc_ops = {
    .proc_open    = continfo_open,
    .proc_read    = seq_read,
    .proc_lseek   = seq_lseek,
    .proc_release = single_release,
};
#else
static const struct file_operations continfo_proc_ops = {
    .owner   = THIS_MODULE,
    .open    = continfo_open,
    .read    = seq_read,
    .llseek  = seq_lseek,
    .release = single_release,
};
#endif

/* ---------------------------------------------------------
 * Init / Exit
 * ---------------------------------------------------------*/

static int __init continfo_init(void)
{
    struct proc_dir_entry *e;

    e = proc_create(PROC_FILENAME, 0444, NULL, &continfo_proc_ops);
    if (!e) {
        pr_err("continfo_so1_202100265: no se pudo crear /proc/%s\n", PROC_FILENAME);
        return -ENOMEM;
    }
    pr_info("continfo_so1_202100265: cargado (/proc/%s)\n", PROC_FILENAME);
    return 0;
}

static void __exit continfo_exit(void)
{
    remove_proc_entry(PROC_FILENAME, NULL);
    pr_info("continfo_so1_202100265: descargado\n");
}

module_init(continfo_init);
module_exit(continfo_exit);