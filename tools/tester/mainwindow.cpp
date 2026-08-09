#include "mainwindow.h"
#include "ui_mainwindow.h"
#include <qdir.h>

MainWindow::MainWindow(QWidget *parent)
    : QMainWindow(parent)
    , ui(new Ui::MainWindow)
{
    ui->setupUi(this);
}

MainWindow::~MainWindow()
{
    delete ui;
}

void MainWindow::on_pushButtonOpen_clicked()
{
    QString path = QFileDialog::getOpenFileName(this, tr("Open ST file"), QString(), tr("ST files (*.st);;All files (*)"));
    if (path.isEmpty())
        return;

    QFile file(path);
    if (!file.open(QIODevice::ReadOnly | QIODevice::Text)) {
        QMessageBox::warning(this, tr("Error"), tr("Failed to open %1").arg(path));
        return;
    }

    ui->plainTextEditSource->setPlainText(QString::fromUtf8(file.readAll()));
}


void MainWindow::on_plainTextEditSource_textChanged()
{
    stSource_ = ui->plainTextEditSource->toPlainText();
}


void MainWindow::on_pushButtonTranslate_clicked()
{
    QString exeName =
#ifdef Q_OS_WIN
        "st2c.exe";
#else
        "st2c";
#endif

    QString exePath = QDir(QCoreApplication::applicationDirPath()).filePath(exeName);

    QProcess process;
    process.start(exePath, QStringList());

    if (!process.waitForStarted()) {
        QMessageBox::warning(this, tr("Error"), tr("Failed to start %1").arg(exePath));
        return;
    }

    process.write(stSource_.toUtf8());
    process.closeWriteChannel();

    if (!process.waitForFinished()) {
        QMessageBox::warning(this, tr("Error"), tr("st2c did not finish"));
        return;
    }

    QByteArray output = process.readAllStandardOutput();
    cResult_ = QString::fromUtf8(output);

    ui->plainTextEditResult->setPlainText(cResult_);
}

