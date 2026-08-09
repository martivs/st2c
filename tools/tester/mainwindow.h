#ifndef MAINWINDOW_H
#define MAINWINDOW_H

#include <QMainWindow>
#include <QCoreApplication>
#include <QDir>
#include <QProcess>
#include <QMessageBox>
#include <QFileDialog>
#include <QFile>


QT_BEGIN_NAMESPACE
namespace Ui {
class MainWindow;
}
QT_END_NAMESPACE

class MainWindow : public QMainWindow
{
    Q_OBJECT

public:
    explicit MainWindow(QWidget *parent = nullptr);
    ~MainWindow() override;

private slots:
    void on_pushButtonOpen_clicked();

    void on_plainTextEditSource_textChanged();

    void on_pushButtonTranslate_clicked();

private:
    Ui::MainWindow *ui;

    QString stSource_;  // содержимое ST
    QString cResult_;   // содержимое C
};
#endif // MAINWINDOW_H
